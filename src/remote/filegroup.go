package remote

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/uploadinfo"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"google.golang.org/protobuf/proto"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// buildFilegroup "builds" a single filegroup target.
//
// Filegroups only rearrange files that already exist, so we calculate their outputs directly from
// those of their sources. We have to upload source files, but we don't need to download anything.
func (c *Client) buildFilegroup(target *core.BuildTarget) (*core.BuildMetadata, error) {
	outs := &pb.Directory{}
	var sources []core.BuildInput
	for _, src := range target.AllSources() {
		if l, ok := src.(core.AnnotatedOutputLabel); ok {
			if err := c.addFilegroupOutputs(outs, target, l.BuildLabel, l.Annotation); err != nil {
				return nil, err
			}
		} else if l, ok := src.Label(); ok {
			if err := c.addFilegroupOutputs(outs, target, l, ""); err != nil {
				return nil, err
			}
		} else {
			sources = append(sources, src)
		}
	}
	if len(sources) > 0 {
		if err := c.addFilegroupSources(outs, target, sources); err != nil {
			return nil, err
		}
	}
	// Sort the outputs so the hash doesn't depend on the order they were collected in.
	slices.SortFunc(outs.Files, func(a, b *pb.FileNode) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(outs.Directories, func(a, b *pb.DirectoryNode) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(outs.Symlinks, func(a, b *pb.SymlinkNode) int { return strings.Compare(a.Name, b.Name) })
	tree := &pb.Tree{Root: outs}
	b, err := proto.Marshal(tree)
	if err != nil {
		return nil, err
	}
	if c.state.TargetHasher != nil {
		c.state.TargetHasher.SetHash(target, c.filegroupHash(outs))
	}
	return &core.BuildMetadata{RemoteOutputs: b, Timestamp: time.Now()}, c.setOutputs(target, tree)
}

// addFilegroupOutputs adds the outputs of a dependency of a filegroup to its outputs.
// If annotation is given then only the named outputs matching it are added.
func (c *Client) addFilegroupOutputs(outs *pb.Directory, target *core.BuildTarget, label core.BuildLabel, annotation string) error {
	for _, provided := range c.state.Graph.TargetOrDie(label).ProvideFor(target) {
		o, err := c.dependencyOutputs(provided)
		if err != nil {
			return err
		}
		if annotation == "" {
			outs.Files = append(outs.Files, o.Files...)
			outs.Directories = append(outs.Directories, o.Directories...)
			outs.Symlinks = append(outs.Symlinks, o.Symlinks...)
			continue
		}
		for _, name := range c.state.Graph.TargetOrDie(provided).NamedOutputs(c.state.Graph, annotation) {
			if d, f, s := findNode(o, name); !appendNode(outs, name, d, f, s) {
				return fmt.Errorf("Missing output from filegroup %s: %s (from %s)", target, name, provided)
			}
		}
	}
	return nil
}

// addFilegroupSources uploads the given source files of a filegroup and adds them to its outputs.
func (c *Client) addFilegroupSources(outs *pb.Directory, target *core.BuildTarget, sources []core.BuildInput) error {
	b := newDirBuilder(c)
	if err := c.uploadBlobs(func(ch chan<- *uploadinfo.Entry) error {
		defer close(ch)
		for _, src := range sources {
			if err := c.addSourceInput(b, ch, target, src); err != nil {
				return err
			}
		}
		// This is needed for the protos of any source directories; it's a bit wasteful for the ones
		// above them, but they're small and this isn't usually many files.
		b.Build(ch)
		return nil
	}); err != nil {
		return err
	}
	for _, src := range sources {
		localPaths := src.LocalPaths(c.state.Graph)
		for i, p := range src.Paths(c.state.Graph) {
			if d, f, s := b.Node(p); !appendNode(outs, localPaths[i], d, f, s) {
				return fmt.Errorf("Missing output from filegroup %s: %s", target, p)
			}
		}
	}
	return nil
}

// filegroupHash returns the output hash for a filegroup. Like outputHash, if there is a single output file
// it's that file's digest (so it can be calculated by hashing the file), otherwise it's the digest of a
// Directory proto of the outputs (not an ActionResult, since filegroups don't have one).
func (c *Client) filegroupHash(outs *pb.Directory) []byte {
	if len(outs.Files) == 1 && len(outs.Directories) == 0 && len(outs.Symlinks) == 0 {
		b, _ := hex.DecodeString(outs.Files[0].Digest.Hash)
		return b
	}
	b, _ := hex.DecodeString(c.digestMessage(outs).Hash)
	return b
}

// downloadFilegroup downloads the outputs of a filegroup. Where possible it does this by downloading its
// dependencies and then constructing the filegroup locally from them, which avoids downloading anything
// twice (and, for filegroups in the same package as their sources, overwriting their outputs).
// Outputs that don't exist locally, or that would need much more than them downloading, are instead
// fetched directly from the CAS.
func (c *Client) downloadFilegroup(target *core.BuildTarget) error {
	var local []core.BuildInput
	remote := map[string]bool{}
	for _, src := range target.AllSources() {
		if names, ok := c.remoteFilegroupOutputs(target, src); ok {
			for _, name := range names {
				remote[name] = true
			}
			continue
		}
		local = append(local, src)
		if l, ok := src.Label(); ok {
			for _, provided := range c.state.Graph.TargetOrDie(l).ProvideFor(target) {
				if err := c.Download(c.state.Graph.TargetOrDie(provided)); err != nil {
					return err
				}
			}
		}
	}
	file, err := core.AcquireExclusiveFileLock(target.BuildLockFile())
	if err != nil {
		return err
	}
	defer core.ReleaseFileLock(file)
	if err := build.MaterialiseFilegroup(c.state, target, local); err != nil {
		return err
	}
	return c.downloadFilegroupNodes(target, remote)
}

// remoteFilegroupOutputs returns the outputs of a filegroup that correspond to one of its sources, if they
// should be fetched directly from the CAS rather than constructed locally from its dependencies.
func (c *Client) remoteFilegroupOutputs(target *core.BuildTarget, src core.BuildInput) ([]string, bool) {
	switch src := src.(type) {
	case core.SubrepoFileLabel:
		// These don't exist locally, so the only place to get them from is the CAS.
		if target.Subrepo.IsRemoteSubrepo() {
			return src.LocalPaths(c.state.Graph), true
		}
	case core.AnnotatedOutputLabel:
		// We only want some of the dependency's outputs, so fetch just those rather than downloading all of it.
		// That doesn't apply if they're in the same package (they're the same files, which we'd clobber) or if
		// the dependency's outputs are already here anyway.
		if target.Label.InSamePackageAs(src.BuildLabel) {
			return nil, false
		}
		var names []string
		for _, provided := range c.state.Graph.TargetOrDie(src.BuildLabel).ProvideFor(target) {
			dep := c.state.Graph.TargetOrDie(provided)
			if c.haveOutputsLocally(dep) {
				return nil, false
			}
			names = append(names, dep.NamedOutputs(c.state.Graph, src.Annotation)...)
		}
		return names, true
	}
	return nil, false
}

// haveOutputsLocally returns true if the outputs of the given target already exist locally and are up to date.
func (c *Client) haveOutputsLocally(target *core.BuildTarget) bool {
	if target.Local {
		return true
	} else if target.IsFilegroup {
		return false // These don't have an action digest to check against (and are cheap to fetch individually anyway)
	}
	return c.outputsExist(target, c.unstampedBuildActionDigests.Get(target.Label))
}

// downloadFilegroupNodes downloads the given outputs of a filegroup from the CAS, skipping any that are already up to date.
func (c *Client) downloadFilegroupNodes(target *core.BuildTarget, names map[string]bool) error {
	if len(names) == 0 {
		return nil
	}
	outs := c.targetOutputs(target.Label)
	outDir := target.OutDir()
	ctx := context.Background()
	files := map[string]*client.TreeOutput{}
	for _, f := range outs.Files {
		if names[f.Name] && !c.nodeUpToDate(filepath.Join(outDir, f.Name), f.Digest) {
			files[f.Name] = &client.TreeOutput{
				Digest:       digest.NewFromProtoUnvalidated(f.Digest),
				Path:         f.Name,
				IsExecutable: f.IsExecutable,
			}
		}
	}
	for _, s := range outs.Symlinks {
		if names[s.Name] {
			if dest, err := os.Readlink(filepath.Join(outDir, s.Name)); err != nil || dest != s.Target {
				files[s.Name] = &client.TreeOutput{Path: s.Name, SymlinkTarget: s.Target}
			}
		}
	}
	var dirs []*pb.DirectoryNode
	for _, d := range outs.Directories {
		if names[d.Name] && !c.nodeUpToDate(filepath.Join(outDir, d.Name), d.Digest) {
			dirs = append(dirs, d)
		}
	}
	if len(files) == 0 && len(dirs) == 0 {
		return nil // Everything's up to date already.
	}
	// Show the target as active while it downloads, as Download does for other targets. Failures aren't
	// logged here since all callers report them already.
	c.state.LogBuildResult(target, core.TargetBuilding, "Downloading...")
	if len(files) > 0 {
		for name := range files {
			if err := fs.RemoveAll(filepath.Join(outDir, name)); err != nil {
				return err
			}
		}
		if _, err := c.client.DownloadOutputs(ctx, files, outDir, c.fileMetadataCache); err != nil {
			return err
		}
		for name, out := range files {
			path := filepath.Join(outDir, name)
			if out.SymlinkTarget != "" {
				c.state.PathHasher.Invalidate(path)
				continue
			}
			// We know the hash of this file now, and must update it in case anything hashed the old one.
			h, _ := hex.DecodeString(out.Digest.Hash)
			c.state.PathHasher.SetHash(path, h)
			c.recordNodeDigest(path, out.Digest.Hash)
		}
	}
	for _, d := range dirs {
		dir := filepath.Join(outDir, d.Name)
		if err := fs.RemoveAll(dir); err != nil {
			return err
		} else if _, _, err := c.client.DownloadDirectory(ctx, digest.NewFromProtoUnvalidated(d.Digest), dir, c.fileMetadataCache); err != nil {
			return err
		}
		c.state.PathHasher.Invalidate(dir)
		c.recordNodeDigest(dir, d.Digest.Hash)
	}
	c.state.LogBuildResult(target, core.TargetBuilt, "Downloaded")
	return nil
}

// nodeUpToDate returns true if the given output of a filegroup exists locally and was downloaded with the given digest.
// This uses its own attribute (rather than the one for targets' action digests) since a filegroup's outputs
// can be the same files as its dependencies'.
func (c *Client) nodeUpToDate(path string, dg *pb.Digest) bool {
	if !fs.PathExists(path) {
		return false
	}
	h, _ := hex.DecodeString(dg.Hash)
	return bytes.Equal(h, fs.ReadAttr(path, nodeXattrName, c.state.XattrsSupported))
}

// recordNodeDigest records the digest of a downloaded output of a filegroup, to be checked by nodeUpToDate later.
func (c *Client) recordNodeDigest(path, hash string) {
	h, _ := hex.DecodeString(hash)
	if err := fs.RecordAttr(path, h, nodeXattrName, c.state.XattrsSupported); err != nil {
		log.Debug("Failed to record digest on %s: %s", path, err)
	}
}

// findNode returns whichever node in the given directory has the given name (or nil for all if none do).
func findNode(dir *pb.Directory, name string) (*pb.DirectoryNode, *pb.FileNode, *pb.SymlinkNode) {
	for _, d := range dir.Directories {
		if d.Name == name {
			return d, nil, nil
		}
	}
	for _, f := range dir.Files {
		if f.Name == name {
			return nil, f, nil
		}
	}
	for _, s := range dir.Symlinks {
		if s.Name == name {
			return nil, nil, s
		}
	}
	return nil, nil, nil
}

// appendNode appends a copy of whichever of the given nodes is non-nil to the given directory, under a new name.
// It returns false if they are all nil.
func appendNode(dir *pb.Directory, name string, d *pb.DirectoryNode, f *pb.FileNode, s *pb.SymlinkNode) bool {
	if d != nil {
		dir.Directories = append(dir.Directories, &pb.DirectoryNode{Name: name, Digest: d.Digest})
	} else if f != nil {
		dir.Files = append(dir.Files, &pb.FileNode{Name: name, Digest: f.Digest, IsExecutable: f.IsExecutable, NodeProperties: f.NodeProperties})
	} else if s != nil {
		dir.Symlinks = append(dir.Symlinks, &pb.SymlinkNode{Name: name, Target: s.Target, NodeProperties: s.NodeProperties})
	} else {
		return false
	}
	return true
}
