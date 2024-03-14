package remote

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/uploadinfo"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/metrics"
	remotefs "github.com/thought-machine/please/src/remote/fs"
)

var downloadErrors = metrics.NewCounter(
	"remote",
	"tree_digest_download_errors_total",
	"Number of times the an error has been seen during a tree digest download",
)

var directoriesRetrieved = metrics.NewCounter(
	"remote",
	"dirs_retrieved_total",
	"Number of directories retrieved from cache",
)

var directoriesDownloaded = metrics.NewCounter(
	"remote",
	"dirs_downloaded_total",
	"Number of directories downloaded from remote",
)

// xattrName is the name we use to record attributes on files.
const xattrName = "user.plz_hash_remote"

// sum calculates a checksum for a byte slice.
func (c *Client) sum(b []byte) []byte {
	h := c.state.PathHasher.NewHash()
	h.Write(b)
	return h.Sum(nil)
}

// targetOutputs returns the outputs for a previously executed target.
// If it has not been executed this returns nil.
func (c *Client) targetOutputs(label core.BuildLabel) *pb.Directory {
	c.outputMutex.RLock()
	defer c.outputMutex.RUnlock()
	return c.outputs[label]
}

// setOutputs sets the outputs for the target, so we can refer to them later by build label from dependent rules. We
// also save the full Tree proto for subrepos, so we can parse and build subrepo targets without downloading the
// sources.
func (c *Client) setOutputs(target *core.BuildTarget, arTree *pb.Tree) error {
	c.outputMutex.Lock()
	defer c.outputMutex.Unlock()

	var outputTree *pb.Tree
	if len(target.OutputDirectories) == 0 {
		outputTree = arTree
	} else {
		o := &pb.Directory{
			Files:          arTree.Root.Files,
			Symlinks:       arTree.Root.Symlinks,
			NodeProperties: arTree.Root.NodeProperties,
			Directories:    make([]*pb.DirectoryNode, 0, len(arTree.Root.Directories)),
		}
		// A filesystem representing the output tree of the action result
		actionResultFS := remotefs.New(c.remoteFSClient, arTree, ".")

		// The action result will contain the raw outputs in their original places. We need to move outputs within
		// `output_dirs` to the output root.
		for _, dirNode := range arTree.Root.Directories {
			if outDir := maybeGetOutDir(dirNode.Name, target.OutputDirectories); outDir != "" {
				// Make sure we add all the outputs to the target
				if err := c.setOutputDirectoryOuts(target, actionResultFS, outDir); err != nil {
					return err
				}

				// Now flatten the contents of the directory into the action result root
				dir := actionResultFS.Dir(digest.NewFromProtoUnvalidated(dirNode.Digest))
				o.Directories = append(o.Directories, dir.Directories...)
				o.Files = append(o.Files, dir.Files...)
				o.Symlinks = append(o.Symlinks, dir.Symlinks...)
			} else {
				o.Directories = append(o.Directories, dirNode)
			}
		}

		outputTree = &pb.Tree{
			Root:     o,
			Children: arTree.Children,
		}
	}

	if target.IsSubrepo {
		c.subrepoTrees[target.Label] = outputTree
	}
	c.outputs[target.Label] = outputTree.Root
	return nil
}

func (c *Client) setOutputsFromMetadata(target *core.BuildTarget, md *core.BuildMetadata) {
	tree := new(pb.Tree)
	if err := proto.Unmarshal(md.RemoteOutputs, tree); err != nil || tree.Root == nil {
		// Something has gone very wrong here if we hit this line, as we should have already checked the proto is valid.
		// tree.Root should never be nil either, but if it is, we will set a nil entry in the outputs map, which will be
		// really hard to debug.
		log.Fatalf("%v: failed to unmarshal output pb.Tree from action metadata: %v", target.Label, err)
	}
	c.setOutputs(target, tree)
}

// outputTree returns a tree representing the outputs of an action result
func (c *Client) outputTree(target *core.BuildTarget, ar *pb.ActionResult) (*pb.Tree, error) {
	o := &pb.Tree{
		Root: &pb.Directory{
			Files:       make([]*pb.FileNode, len(ar.OutputFiles)),
			Directories: make([]*pb.DirectoryNode, 0, len(ar.OutputDirectories)),
			Symlinks:    make([]*pb.SymlinkNode, len(ar.OutputFileSymlinks)+len(ar.OutputDirectorySymlinks)),
		},
	}
	// N.B. At this point the various things we stick into this Directory proto can be in
	//      subdirectories. This is not how a Directory proto is meant to work but it makes things
	//      a lot easier for us to handle (since it is impossible to merge two DirectoryNode protos
	//      without downloading their respective Directory protos). Later on we sort this out in
	//      uploadInputDir.
	for i, f := range ar.OutputFiles {
		o.Root.Files[i] = &pb.FileNode{
			Name:         target.GetRealOutput(f.Path),
			Digest:       f.Digest,
			IsExecutable: f.IsExecutable,
		}
	}
	for _, d := range ar.OutputDirectories {
		tree := &pb.Tree{}
		if _, err := c.client.ReadProto(context.Background(), digest.NewFromProtoUnvalidated(d.TreeDigest), tree); err != nil {
			downloadErrors.Inc()
			return nil, wrap(err, "Downloading tree digest for %s [%s]", d.Path, d.TreeDigest.Hash)
		}
		o.Children = append(o.Children, append(tree.Children, tree.Root)...)
		o.Root.Directories = append(o.Root.Directories, &pb.DirectoryNode{
			Name:   target.GetRealOutput(d.Path),
			Digest: c.digestMessage(tree.Root),
		})
	}
	for i, s := range append(ar.OutputFileSymlinks, ar.OutputDirectorySymlinks...) {
		o.Root.Symlinks[i] = &pb.SymlinkNode{
			Name:   target.GetRealOutput(s.Path),
			Target: s.Target,
		}
	}
	return o, nil
}

// setOutputDirectoryOuts sets the output on the target based on the outputs in the action result
func (c *Client) setOutputDirectoryOuts(target *core.BuildTarget, actionResultFS *remotefs.CASFileSystem, outDir core.OutputDirectory) error {
	outDirFS := actionResultFS.ChangeDir(outDir.Dir())
	if outDir.ShouldAddFiles() {
		err := iofs.WalkDir(outDirFS, ".", func(path string, d iofs.DirEntry, err error) error {
			if !d.IsDir() {
				target.AddOutput(path)
			}
			return nil
		})
		return err
	}

	entries, err := iofs.ReadDir(outDirFS, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		target.AddOutput(e.Name())
	}
	return nil
}

// readDirectory reads a Directory proto, possibly using a local cache, otherwise going to the remote
func (c *Client) readDirectory(dg *pb.Digest) (*pb.Directory, error) {
	if dir, present := c.directories.Load(dg.Hash); present {
		directoriesRetrieved.Inc()
		return dir.(*pb.Directory), nil
	}
	dir := &pb.Directory{}
	_, err := c.client.ReadProto(context.Background(), digest.NewFromProtoUnvalidated(dg), dir)
	directoriesDownloaded.Inc()
	return dir, err
}

// maybeGetOutDir will get the output directory based on the directory provided. If there's no matching directory, this
// will return an empty string indicating that that action output was not an output directory.
func maybeGetOutDir(dir string, outDirs []core.OutputDirectory) core.OutputDirectory {
	for _, outDir := range outDirs {
		if dir == outDir.Dir() {
			return outDir
		}
	}
	return ""
}

// digestMessage calculates the digest of a protobuf in SHA-256 mode.
func (c *Client) digestMessage(msg proto.Message) *pb.Digest {
	// Can't use NewFromMessage because remote-apis-sdks is still using the older interface.
	blob, _ := proto.Marshal(msg)
	return digest.NewFromBlob(blob).ToProto()
}

// wrapActionErr wraps an error with information about the action related to it.
func (c *Client) wrapActionErr(err error, actionDigest *pb.Digest) error {
	if err == nil || c.state.Config.Remote.DisplayURL == "" {
		return err
	}
	return wrap(err, "Action URL: %s/action/%s/%s/%d/\n", c.state.Config.Remote.DisplayURL, c.state.Config.Remote.Instance, actionDigest.Hash, actionDigest.SizeBytes)
}

// actionURL returns a URL to the browser for a remote action, if the display URL is configured.
// If prefix is true then it is surrounded by "(action: %s)".
func (c *Client) actionURL(digest *pb.Digest, prefix bool) string {
	if c.state.Config.Remote.DisplayURL == "" {
		return ""
	}
	s := fmt.Sprintf("%s/action/%s/%s/%d/", c.state.Config.Remote.DisplayURL, c.state.Config.Remote.Instance, digest.Hash, digest.SizeBytes)
	if prefix {
		s = " (action: " + s + ")"
	}
	return s
}

// locallyCacheResults stores the actionresult for an action in the local (usually dir) cache.
func (c *Client) locallyCacheResults(target *core.BuildTarget, dg *pb.Digest, metadata *core.BuildMetadata) {
	if c.state.Cache == nil {
		return
	}

	if err := c.mdStore.storeMetadata(dg.Hash, metadata); err != nil {
		log.Warningf("Failed to store build metadata for target %s: %v", target.Label, err)
	}
}

// retrieveLocalResults retrieves locally cached results for a target if possible.
// Note that this does not handle any file data, only the actionresult metadata.
func (c *Client) retrieveLocalResults(target *core.BuildTarget, digest *pb.Digest) (*core.BuildMetadata, *pb.ActionResult) {
	if c.state.Cache != nil {
		metadata, err := c.mdStore.retrieveMetadata(digest.Hash)
		if err != nil {
			log.Warningf("Failed to retrieve stored metadata for target %s, %v", target.Label, err)
		}
		// Empty filegroups have empty action results
		isEmptyFilegroup := target.IsFilegroup && len(target.AllSources()) == 0
		if metadata != nil && (len(metadata.RemoteAction) > 0 || isEmptyFilegroup) {
			ar := &pb.ActionResult{}
			outputs := &pb.Tree{}
			if err := proto.Unmarshal(metadata.RemoteAction, ar); err != nil {
				return nil, nil
			}
			if err := proto.Unmarshal(metadata.RemoteOutputs, outputs); err != nil {
				return nil, nil
			}

			if outputs.Root == nil {
				return nil, nil
			}

			return metadata, ar
		}
	}
	return nil, nil
}

// outputsExist returns true if the outputs for this target exist and are up to date.
func (c *Client) outputsExist(target *core.BuildTarget, digest *pb.Digest) bool {
	hash, _ := hex.DecodeString(digest.Hash)
	for _, out := range target.FullOutputs() {
		if !bytes.Equal(hash, fs.ReadAttr(out, xattrName, c.state.XattrsSupported)) {
			return false
		}
	}
	return true
}

// recordAttrs sets the xattrs on output files which we will use in outputsExist in future runs.
func (c *Client) recordAttrs(target *core.BuildTarget, digest *pb.Digest) {
	hash, _ := hex.DecodeString(digest.Hash)
	for _, out := range target.FullOutputs() {
		fs.RecordAttr(out, hash, xattrName, c.state.XattrsSupported)
	}
}

// mustMarshal encodes a message to a binary string.
func mustMarshal(msg proto.Message) []byte {
	b, err := proto.Marshal(msg)
	if err != nil {
		// Not really sure if there is a valid possibility to bring us here (given that
		// the messages in question have no required fields) so assume it won't happen :)
		log.Fatalf("Failed to marshal message: %s", err)
	}
	return b
}

// lessThan returns true if the given semver instance is less than another one.
func lessThan(a, b *semver.SemVer) bool {
	if a.Major < b.Major {
		return true
	} else if a.Major > b.Major {
		return false
	} else if a.Minor < b.Minor {
		return true
	} else if a.Minor > b.Minor {
		return false
	} else if a.Patch < b.Patch {
		return true
	} else if a.Patch > b.Patch {
		return false
	}
	return a.Prerelease < b.Prerelease
}

// printVer pretty-prints a semver message.
// The default stringing of them is so bad as to be completely unreadable.
func printVer(v *semver.SemVer) string {
	msg := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		msg += "-" + v.Prerelease
	}
	return msg
}

// IsNotFound returns true if a given error is a "not found" error (which may be treated
// differently, for example if trying to retrieve artifacts that may not be there).
func IsNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

// hasChild returns true if a Directory has a child directory by the given name.
func hasChild(dir *pb.Directory, child string) bool {
	for _, d := range dir.Directories {
		if d.Name == child {
			return true
		}
	}
	return false
}

// convertError converts a single google.rpc.Status message into a Go error
func convertError(err *rpcstatus.Status) error {
	if err.Code == int32(codes.OK) {
		return nil
	}

	if err.Code == int32(codes.DeadlineExceeded) {
		return context.DeadlineExceeded
	}

	msg := status.ErrorProto(err)
	for _, detail := range err.Details {
		msg = fmt.Errorf("%s %s", msg, detail.Value)
	}
	return msg
}

// wrap wraps a grpc error in an additional description, but retains its code.
func wrap(err error, msg string, args ...interface{}) error {
	s, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf(fmt.Sprintf(msg, args...) + ": " + err.Error())
	}
	return status.Errorf(s.Code(), fmt.Sprintf(msg, args...)+": "+s.Message())
}

// timeout returns either a build or test timeout from a target.
func timeout(target *core.BuildTarget, test bool) time.Duration {
	if test {
		return target.Test.Timeout
	}
	return target.BuildTimeout
}

// A dirBuilder is for helping build up a tree of Directory protos.
//
// This is pretty awkward; we need to recursively build a whole set of directories
// which does not match up to how we represent it (which is a series of files, with
// no corresponding directories, that are not usefully ordered for this purpose).
// We also need to handle the case of existing targets where we already know the
// directory structure but may not have the files physically on disk.
type dirBuilder struct {
	c    *Client
	root *pb.Directory
	dirs map[string]*pb.Directory
}

func newDirBuilder(c *Client) *dirBuilder {
	root := &pb.Directory{}
	return &dirBuilder{
		dirs: map[string]*pb.Directory{
			".": root, // Ensure the root is in there
			"":  root, // Some things might try to name it this way
		},
		root: root,
		c:    c,
	}
}

// Dir ensures the given directory exists, and constructs any necessary parents.
func (b *dirBuilder) Dir(name string) *pb.Directory {
	return b.dir(name, "")
}

func (b *dirBuilder) dir(dir, child string) *pb.Directory {
	if dir == "." || dir == "/" {
		return b.root
	}
	dir = strings.TrimSuffix(dir, "/")
	d, present := b.dirs[dir]
	if !present {
		d = &pb.Directory{}
		b.dirs[dir] = d
		dir, base := filepath.Split(dir)
		b.dir(dir, base)
	}
	// TODO(peterebden): The linear scan in hasChild is a bit suboptimal, we should
	//                   really use the dirs map to determine this.
	if child != "" && !hasChild(d, child) {
		d.Directories = append(d.Directories, &pb.DirectoryNode{Name: child})
	}
	return d
}

// Build "builds" the directory. It calculates the digests of all the items in the directory tree, and returns the root
// directory. If ch is non-nil, it will upload the directory protos to ch. Build doesn't upload any of the actual files
// in the directory tree, just the protos.
func (b *dirBuilder) Build(ch chan<- *uploadinfo.Entry) *pb.Directory {
	// Upload the directory structure
	b.walk(".", ch)
	return b.root
}

// Node returns either the file or directory corresponding to the given path (or nil for both if not found)
func (b *dirBuilder) Node(name string) (*pb.DirectoryNode, *pb.FileNode) {
	dir := b.Dir(filepath.Dir(name))
	base := filepath.Base(name)
	for _, d := range dir.Directories {
		if d.Name == base {
			return d, nil
		}
	}
	for _, f := range dir.Files {
		if f.Name == base {
			return nil, f
		}
	}
	return nil, nil
}

// Tree returns the tree rooted at a given directory name.
// It does not calculate digests or upload, so call Build beforehand if that is needed.
func (b *dirBuilder) Tree(root string) *pb.Tree {
	d := b.dir(root, "")
	tree := &pb.Tree{Root: d}
	b.tree(tree, root, d)
	return tree
}

func (b *dirBuilder) tree(tree *pb.Tree, root string, dir *pb.Directory) {
	tree.Children = append(tree.Children, dir)
	for _, d := range dir.Directories {
		name := filepath.Join(root, d.Name)
		b.tree(tree, name, b.dirs[name])
	}
}

// Walk walks the directory tree calculating the digest. If ch is non-nil, it will also upload the direcory protos.
// Walk does not upload the actual files in the tree, just the tree structure.
func (b *dirBuilder) walk(name string, ch chan<- *uploadinfo.Entry) *pb.Digest {
	dir := b.dirs[name]
	for _, d := range dir.Directories {
		if d.Digest == nil { // It's not nil if we're reusing outputs from an earlier call.
			d.Digest = b.walk(filepath.Join(name, d.Name), ch)
		}
	}
	// The protocol requires that these are sorted into lexicographic order. Not all servers
	// necessarily care, but some do, and we should be compliant.
	files := dir.Files
	dirs := dir.Directories
	syms := dir.Symlinks
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(syms, func(i, j int) bool { return syms[i].Name < syms[j].Name })

	// Ensure there are not duplicates in these slices.
	last := ""
	dir.Files = files[:0]
	for _, f := range files {
		if f.Name != last {
			dir.Files = append(dir.Files, f)
			last = f.Name
		}
	}
	dir.Directories = dirs[:0]
	for _, d := range dirs {
		if d.Name != last {
			dir.Directories = append(dir.Directories, d)
			last = d.Name
		}
	}
	dir.Symlinks = syms[:0]
	for _, s := range syms {
		if s.Name != last {
			dir.Symlinks = append(dir.Symlinks, s)
			last = s.Name
		}
	}

	entry, _ := uploadinfo.EntryFromProto(dir)
	if ch != nil {
		ch <- entry
	}
	return entry.Digest.ToProto()
}

// convertPlatform converts the platform entries from the config into a Platform proto.
func convertPlatform(properties []string) *pb.Platform {
	platform := &pb.Platform{}
	for _, p := range properties {
		if parts := strings.SplitN(p, "=", 2); len(parts) == 2 {
			platform.Properties = append(platform.Properties, &pb.Platform_Property{
				Name:  parts[0],
				Value: parts[1],
			})
		} else {
			log.Warning("Invalid platform property setting %s; will ignore", p)
		}
	}
	return platform
}

// targetPlatformProperties returns the platform properties for a target, including any global ones.
func (c *Client) targetPlatformProperties(target *core.BuildTarget) *pb.Platform {
	labels := target.PrefixedLabels("remote-platform-property:")
	if len(labels) == 0 {
		return c.platform
	}
	platform := convertPlatform(labels)
	platform.Properties = append(platform.Properties, c.platform.Properties...)
	return platform
}

// removeOutputs removes all outputs for a target.
func removeOutputs(target *core.BuildTarget) error {
	outDir := target.OutDir()
	for _, out := range target.Outputs() {
		if err := os.RemoveAll(filepath.Join(outDir, out)); err != nil {
			return fmt.Errorf("Failed to remove output for %s: %s", target, err)
		}
	}
	return nil
}

// subresourceIntegrity returns a string corresponding to a target's hashes in the Subresource Integrity format.
func subresourceIntegrity(target *core.BuildTarget) string {
	ret := make([]string, len(target.Hashes))
	for i, h := range target.Hashes {
		ret[i] = reencodeSRI(target, h)
	}
	return strings.Join(ret, " ")
}

// reencodeSRI re-encodes a hash from the hex format we use to base64-encoded.
func reencodeSRI(target *core.BuildTarget, h string) string {
	if idx := strings.LastIndexByte(h, ':'); idx != -1 {
		h = h[idx+1:]
	}
	// TODO(peterebden): we should validate at parse time that these are sensible.
	b, _ := hex.DecodeString(h)
	h = base64.StdEncoding.EncodeToString(b)
	if len(b) == sha256.Size {
		return "sha256-" + h
	} else if len(b) == sha1.Size {
		return "sha1-" + h
	}
	log.Warning("Hash string of unknown type on %s: %s", target, h)
	return h
}

// dialOpts returns a set of dial options to apply based on the config.
func (c *Client) dialOpts() ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{
		grpc.WithStatsHandler(c.stats),
		// Set an arbitrarily large (400MB) max message size so it isn't a limitation.
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(419430400)),
		grpc.WithChainUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithChainStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	if c.state.Config.Remote.TokenFile == "" {
		return opts, nil
	}
	token, err := os.ReadFile(c.state.Config.Remote.TokenFile)
	if err != nil {
		return opts, fmt.Errorf("Failed to load token from file: %s", err)
	}
	return append(opts, grpc.WithPerRPCCredentials(preSharedToken(string(token)))), nil
}

// outputHash returns an output hash for a target. If it has a single output it's the hash
// of that output, otherwise it's the hash of the whole thing.
// The special-casing is important to make remote_file hash properly (also so you can
// calculate it manually by sha256sum'ing the file).
func (c *Client) outputHash(ar *pb.ActionResult) string {
	if len(ar.OutputFiles) == 1 && len(ar.OutputDirectories) == 0 && len(ar.OutputFileSymlinks) == 0 && len(ar.OutputDirectorySymlinks) == 0 {
		return ar.OutputFiles[0].Digest.Hash
	}
	return c.digestMessage(ar).Hash
}

// preSharedToken returns a gRPC credential provider for a pre-shared token.
func preSharedToken(token string) tokenCredProvider {
	return tokenCredProvider{
		"authorization": "Bearer " + strings.TrimSpace(token),
	}
}

type tokenCredProvider map[string]string

func (cred tokenCredProvider) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return cred, nil
}

func (cred tokenCredProvider) RequireTransportSecurity() bool {
	return false // Allow these to be provided over an insecure channel; this facilitates e.g. service meshes like Istio.
}

// contextWithMetadata returns a context with metadata corresponding to the given build target.
func (c *Client) contextWithMetadata(target *core.BuildTarget) context.Context {
	const key = "build.bazel.remote.execution.v2.requestmetadata-bin" // as defined by the proto
	b, _ := proto.Marshal(&pb.RequestMetadata{
		ActionId:                target.Label.String(),
		CorrelatedInvocationsId: c.buildID,
		ToolDetails: &pb.ToolDetails{
			ToolName:    "please",
			ToolVersion: core.PleaseVersion,
		},
	})
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(key, string(b)))
}
