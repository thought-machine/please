package remote

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
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

// setOutputs sets the outputs for a previously executed target.
func (c *Client) setOutputs(label core.BuildLabel, ar *pb.ActionResult) error {
	o := &pb.Directory{
		Files:       make([]*pb.FileNode, len(ar.OutputFiles)),
		Directories: make([]*pb.DirectoryNode, len(ar.OutputDirectories)),
		Symlinks:    make([]*pb.SymlinkNode, len(ar.OutputFileSymlinks)+len(ar.OutputDirectorySymlinks)),
	}
	// N.B. At this point the various things we stick into this Directory proto can be in
	//      subdirectories. This is not how a Directory proto is meant to work but it makes things
	//      a lot easier for us to handle (since it is impossible to merge two DirectoryNode protos
	//      without downloading their respective Directory protos). Later on we sort this out in
	//      uploadInputDir.
	for i, f := range ar.OutputFiles {
		o.Files[i] = &pb.FileNode{
			Name:         f.Path,
			Digest:       f.Digest,
			IsExecutable: f.IsExecutable,
		}
	}
	for i, d := range ar.OutputDirectories {
		tree := &pb.Tree{}
		if err := c.client.ReadProto(context.Background(), digest.NewFromProtoUnvalidated(d.TreeDigest), tree); err != nil {
			return wrap(err, "Downloading tree digest for %s [%s]", d.Path, d.TreeDigest.Hash)
		}
		o.Directories[i] = &pb.DirectoryNode{
			Name:   d.Path,
			Digest: c.digestMessage(tree.Root),
		}
	}
	for i, s := range append(ar.OutputFileSymlinks, ar.OutputDirectorySymlinks...) {
		o.Symlinks[i] = &pb.SymlinkNode{
			Name:   s.Path,
			Target: s.Target,
		}
	}
	c.outputMutex.Lock()
	defer c.outputMutex.Unlock()
	c.outputs[label] = o
	return nil
}

// digestMessage calculates the digest of a proto message as described in the
// Digest message's comments.
func (c *Client) digestMessage(msg proto.Message) *pb.Digest {
	digest, _ := c.digestMessageContents(msg)
	return digest
}

// digestMessageContents is like DigestMessage but returns the serialised contents as well.
func (c *Client) digestMessageContents(msg proto.Message) (*pb.Digest, []byte) {
	b := mustMarshal(msg)
	return c.digestBlob(b), b
}

// digestBlob digests a byteslice and returns the proto for it.
func (c *Client) digestBlob(b []byte) *pb.Digest {
	sum := c.sum(b)
	return &pb.Digest{
		Hash:      hex.EncodeToString(sum[:]),
		SizeBytes: int64(len(b)),
	}
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
func (c *Client) locallyCacheResults(target *core.BuildTarget, digest *pb.Digest, metadata *core.BuildMetadata, ar *pb.ActionResult) {
	if c.state.Cache == nil {
		return
	}
	data, _ := proto.Marshal(ar)
	metadata.RemoteAction = data
	key, _ := hex.DecodeString(digest.Hash)
	c.state.Cache.Store(target, key, metadata, nil)
}

// retrieveLocalResults retrieves locally cached results for a target if possible.
// Note that this does not handle any file data, only the actionresult metadata.
func (c *Client) retrieveLocalResults(target *core.BuildTarget, digest *pb.Digest) (*core.BuildMetadata, *pb.ActionResult) {
	if c.state.Cache != nil {
		key, _ := hex.DecodeString(digest.Hash)
		if metadata := c.state.Cache.Retrieve(target, key, nil); metadata != nil && len(metadata.RemoteAction) > 0 {
			ar := &pb.ActionResult{}
			if err := proto.Unmarshal(metadata.RemoteAction, ar); err == nil {
				if err := c.setOutputs(target.Label, ar); err == nil {
					return metadata, ar
				}
			}
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

// toTimestamp converts a time.Time into a protobuf timestamp
func toTimestamp(t time.Time) *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

// toTime converts a protobuf timestamp into a time.Time.
// It's like the ptypes one but we ignore errors (we don't generally care that much)
func toTime(ts *timestamp.Timestamp) time.Time {
	t, _ := ptypes.Timestamp(ts)
	return t
}

// extraPerms returns any additional permission bits we should apply for this file.
func extraPerms(file *pb.OutputFile) os.FileMode {
	if file.IsExecutable {
		return 0111
	}
	return 0
}

// extraFilePerms returns any additional permission bits we should apply for this file.
func extraFilePerms(file *pb.FileNode) os.FileMode {
	if file.IsExecutable {
		return 0111
	}
	return 0
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

// exhaustChannel reads and discards all messages on the given channel.
func exhaustChannel(ch <-chan *blob) {
	for range ch {
	}
}

// convertError converts a single google.rpc.Status message into a Go error
func convertError(err *rpcstatus.Status) error {
	if err.Code == int32(codes.OK) {
		return nil
	}
	msg := fmt.Errorf("%s", err.Message)
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
		return target.TestTimeout
	}
	return target.BuildTimeout
}

// outputs returns the outputs of a target, split arbitrarily and inaccurately
// into files and directories.
// After some discussion we are hoping that servers are permissive about this if
// we get it wrong; we prefer to make an effort though as a minor nicety.
func outputs(target *core.BuildTarget) (files, dirs []string) {
	outs := target.Outputs()
	files = make([]string, 0, len(outs))
	for _, out := range outs {
		out = target.GetTmpOutput(out)
		if !strings.ContainsRune(path.Base(out), '.') && !strings.HasSuffix(out, "file") && !target.IsBinary {
			dirs = append(dirs, out)
		} else {
			files = append(files, out)
		}
	}
	return files, dirs
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
		dir, base := path.Split(dir)
		b.dir(dir, base)
	}
	// TODO(peterebden): The linear scan in hasChild is a bit suboptimal, we should
	//                   really use the dirs map to determine this.
	if child != "" && !hasChild(d, child) {
		d.Directories = append(d.Directories, &pb.DirectoryNode{Name: child})
	}
	return d
}

// Root returns the root directory, calculates the digests of all others and uploads them
// if the given channel is not nil.
func (b *dirBuilder) Root(ch chan<- *blob) *pb.Directory {
	b.dfs(".", ch)
	return b.root
}

// Node returns either the file or directory corresponding to the given path (or nil for both if not found)
func (b *dirBuilder) Node(name string) (*pb.DirectoryNode, *pb.FileNode) {
	dir := b.Dir(path.Dir(name))
	base := path.Base(name)
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
// It does not calculate digests or upload, so call Root beforehand if that is needed.
func (b *dirBuilder) Tree(ch chan<- *blob, root string) *pb.Tree {
	d := b.dir(root, "")
	tree := &pb.Tree{Root: d}
	b.tree(tree, root, d)
	return tree
}

func (b *dirBuilder) tree(tree *pb.Tree, root string, dir *pb.Directory) {
	tree.Children = append(tree.Children, dir)
	for _, d := range dir.Directories {
		name := path.Join(root, d.Name)
		b.tree(tree, name, b.dirs[name])
	}
}

func (b *dirBuilder) dfs(name string, ch chan<- *blob) *pb.Digest {
	dir := b.dirs[name]
	for _, d := range dir.Directories {
		if d.Digest == nil { // It's not nil if we're reusing outputs from an earlier call.
			d.Digest = b.dfs(path.Join(name, d.Name), ch)
		}
	}
	digest, contents := b.c.digestMessageContents(dir)
	if ch != nil {
		ch <- &blob{
			Digest: digest,
			Data:   contents,
		}
	}
	return digest
}

// convertPlatform converts the platform entries from the config into a Platform proto.
func convertPlatform(config *core.Configuration) *pb.Platform {
	platform := &pb.Platform{}
	for _, p := range config.Remote.Platform {
		if parts := strings.SplitN(p, "=", 2); len(parts) == 2 {
			platform.Properties = append(platform.Properties, &pb.Platform_Property{
				Name:  parts[0],
				Value: parts[1],
			})
		} else {
			log.Warning("Invalid config setting in remote.platform %s; will ignore", p)
		}
	}
	return platform
}

// removeOutputs removes all outputs for a target.
func removeOutputs(target *core.BuildTarget) error {
	outDir := target.OutDir()
	for _, out := range target.Outputs() {
		if err := os.RemoveAll(path.Join(outDir, out)); err != nil {
			return fmt.Errorf("Failed to remove output for %s: %s", target, err)
		}
	}
	return nil
}

// subresourceIntegrity returns a string corresponding to a target's hashes in the Subresource Integrity format.
func subresourceIntegrity(hashes []string) string {
	ret := make([]string, len(hashes))
	for i, h := range hashes {
		ret[i] = reencodeSRI(h)
	}
	return strings.Join(ret, " ")
}

// reencodeSRI re-encodes a hash from the hex format we use to base64-encoded.
func reencodeSRI(h string) string {
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
	log.Warning("Hash string of unknown type: %s", h)
	return h
}

// updateHashFilename updates an output filename for a hash_filegroup.
func updateHashFilename(name string, digest *pb.Digest) string {
	ext := path.Ext(name)
	before := name[:len(name)-len(ext)]
	b, _ := hex.DecodeString(digest.Hash)
	return before + "-" + base64.RawURLEncoding.EncodeToString(b) + ext
}
