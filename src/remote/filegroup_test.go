package remote

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// putBlob stores a blob on the test server and returns its digest.
func putBlob(b []byte) *pb.Digest {
	dg := digest.NewFromBlob(b)
	server.blobs[dg.Hash] = b
	return dg.ToProto()
}

// putProto stores a proto on the test server and returns its digest.
func putProto(msg proto.Message) *pb.Digest {
	return putBlob(mustMarshal(msg))
}

// A remoteDep describes the contents of a remotely built dependency of a filegroup.
type remoteDep struct {
	Target                   *core.BuildTarget
	File, Nested             []byte
	FileDigest, NestedDigest *pb.Digest
	// DirDigest is the digest of the root Directory proto of the output directory.
	DirDigest *pb.Digest
}

// buildRemoteDep builds a target remotely that outputs a single file (<name>_out.txt) and a
// directory (<name>_dir) containing a subdirectory, so the output tree is more than one level deep.
func buildRemoteDep(t *testing.T, c *Client, label core.BuildLabel) *remoteDep {
	t.Helper()
	dep := &remoteDep{
		File:   []byte("file output of " + label.String()),
		Nested: []byte("nested output of " + label.String()),
	}
	dep.FileDigest = putBlob(dep.File)
	dep.NestedDigest = putBlob(dep.Nested)
	sub := &pb.Directory{Files: []*pb.FileNode{{Name: "nested.txt", Digest: dep.NestedDigest}}}
	root := &pb.Directory{Directories: []*pb.DirectoryNode{{Name: "sub", Digest: putProto(sub)}}}
	dep.DirDigest = putProto(root)
	treeDigest := putProto(&pb.Tree{Root: root, Children: []*pb.Directory{sub}})

	server.mockActionResult = &pb.ActionResult{
		OutputFiles:       []*pb.OutputFile{{Path: label.Name + "_out.txt", Digest: dep.FileDigest}},
		OutputDirectories: []*pb.OutputDirectory{{Path: label.Name + "_dir", TreeDigest: treeDigest}},
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			Worker:                      "kev",
			QueuedTimestamp:             timestamppb.Now(),
			ExecutionStartTimestamp:     timestamppb.Now(),
			ExecutionCompletedTimestamp: timestamppb.Now(),
		},
	}
	dep.Target = core.NewBuildTarget(label)
	dep.Target.AddOutput(label.Name + "_out.txt")
	dep.Target.AddOutput(label.Name + "_dir")
	// Doesn't get executed, but makes the action unique to this target.
	dep.Target.Command = "echo " + label.String()
	c.state.Graph.AddTarget(dep.Target)
	_, err := c.Build(dep.Target)
	require.NoError(t, err)
	dep.Target.SetState(core.BuiltRemotely)
	return dep
}

// newFilegroup creates a filegroup with the given sources.
func newFilegroup(c *Client, label core.BuildLabel, srcs ...core.BuildInput) *core.BuildTarget {
	fg := core.NewBuildTarget(label)
	fg.IsFilegroup = true
	for _, src := range srcs {
		fg.AddSource(src)
	}
	c.state.Graph.AddTarget(fg)
	return fg
}

// assertNoRemoteActions asserts that nothing touched the action cache or execution service.
func assertNoRemoteActions(t *testing.T) {
	t.Helper()
	assert.Zero(t, server.calls.GetActionResult.Load(), "should not look up any action results")
	assert.Zero(t, server.calls.UpdateActionResult.Load(), "should not write any action results")
	assert.Zero(t, server.calls.Execute.Load(), "should not execute anything")
}

func findFile(dir *pb.Directory, name string) *pb.FileNode {
	for _, f := range dir.Files {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func findDir(dir *pb.Directory, name string) *pb.DirectoryNode {
	for _, d := range dir.Directories {
		if d.Name == name {
			return d
		}
	}
	return nil
}

func TestFilegroupBuildDoesNotUseActions(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	dep := buildRemoteDep(t, c, core.BuildLabel{PackageName: "package2", Name: "fg_no_actions_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_no_actions"},
		dep.Target.Label, core.FileLabel{File: "src1.txt", Package: "package"})

	server.calls.Reset()
	_, err := c.Build(fg)
	require.NoError(t, err)
	assertNoRemoteActions(t)
	assert.Zero(t, server.calls.BlobReads.Load(), "should not need to read anything from the CAS")
}

func TestFilegroupOutputs(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	dep := buildRemoteDep(t, c, core.BuildLabel{PackageName: "package2", Name: "fg_outputs_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_outputs"},
		dep.Target.Label, core.FileLabel{File: "src1.txt", Package: "package"})
	_, err := c.Build(fg)
	require.NoError(t, err)

	outs := c.targetOutputs(fg.Label)
	require.NotNil(t, outs)
	f := findFile(outs, "fg_outputs_dep_out.txt")
	require.NotNil(t, f)
	assert.Equal(t, dep.FileDigest.Hash, f.Digest.Hash)
	d := findDir(outs, "fg_outputs_dep_dir")
	require.NotNil(t, d)
	assert.Equal(t, dep.DirDigest.Hash, d.Digest.Hash)

	src, err := os.ReadFile("package/src1.txt")
	require.NoError(t, err)
	srcDigest := digest.NewFromBlob(src)
	f = findFile(outs, "src1.txt")
	require.NotNil(t, f)
	assert.Equal(t, srcDigest.Hash, f.Digest.Hash)
	// Dependent actions refer to this by digest, so it must have been uploaded.
	assert.Equal(t, src, server.blobs[srcDigest.Hash])
}

func TestFilegroupConsumerInputRoot(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	dep := buildRemoteDep(t, c, core.BuildLabel{PackageName: "package2", Name: "fg_consumer_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_consumer"},
		dep.Target.Label, core.FileLabel{File: "src1.txt", Package: "package"})
	_, err := c.Build(fg)
	require.NoError(t, err)

	consumer := core.NewBuildTarget(core.BuildLabel{PackageName: "consumer", Name: "consumer"})
	consumer.AddSource(fg.Label)
	consumer.AddOutput("consumer.txt")
	c.state.Graph.AddTarget(consumer)
	b, err := c.uploadInputDir(nil, consumer, false)
	require.NoError(t, err)
	b.Build(nil)

	// The filegroup's outputs appear in its package, not the package of its dependency.
	assert.Nil(t, b.dirs["package2"])
	pkg := b.dirs["package"]
	require.NotNil(t, pkg)
	assert.NotNil(t, findFile(pkg, "fg_consumer_dep_out.txt"))
	assert.NotNil(t, findFile(pkg, "src1.txt"))
	d := findDir(pkg, "fg_consumer_dep_dir")
	require.NotNil(t, d)
	assert.Equal(t, dep.DirDigest.Hash, d.Digest.Hash)
}

func TestFilegroupNamedOutputs(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	a := putBlob([]byte("named output a"))
	b := putBlob([]byte("named output b"))
	server.mockActionResult = &pb.ActionResult{
		OutputFiles: []*pb.OutputFile{
			{Path: "fg_named_a.txt", Digest: a},
			{Path: "fg_named_b.txt", Digest: b},
		},
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			QueuedTimestamp:             timestamppb.Now(),
			ExecutionStartTimestamp:     timestamppb.Now(),
			ExecutionCompletedTimestamp: timestamppb.Now(),
		},
	}
	dep := core.NewBuildTarget(core.BuildLabel{PackageName: "package2", Name: "fg_named_dep"})
	dep.AddNamedOutput("a", "fg_named_a.txt")
	dep.AddNamedOutput("b", "fg_named_b.txt")
	dep.Command = "echo named"
	c.state.Graph.AddTarget(dep)
	_, err := c.Build(dep)
	require.NoError(t, err)

	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_named"},
		core.AnnotatedOutputLabel{BuildLabel: dep.Label, Annotation: "a"})
	server.calls.Reset()
	_, err = c.Build(fg)
	require.NoError(t, err)
	assertNoRemoteActions(t)

	outs := c.targetOutputs(fg.Label)
	require.NotNil(t, outs)
	require.Len(t, outs.Files, 1)
	assert.Equal(t, "fg_named_a.txt", outs.Files[0].Name)
	assert.Equal(t, a.Hash, outs.Files[0].Digest.Hash)
}

func TestEmptyFilegroup(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_empty"})
	server.calls.Reset()
	_, err := c.Build(fg)
	require.NoError(t, err)
	assertNoRemoteActions(t)
	outs := c.targetOutputs(fg.Label)
	require.NotNil(t, outs)
	assert.Empty(t, outs.Files)
	assert.Empty(t, outs.Directories)
}

func TestFilegroupDownloadReusesDependencyOutputs(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	build.Init(c.state)
	dep := buildRemoteDep(t, c, core.BuildLabel{PackageName: "package2", Name: "fg_download_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_download"},
		dep.Target.Label, core.FileLabel{File: "src1.txt", Package: "package"})
	_, err := c.Build(fg)
	require.NoError(t, err)
	require.NoError(t, c.Download(dep.Target))

	// The dependency's outputs are already here, so the filegroup shouldn't fetch them again.
	server.calls.Reset()
	require.NoError(t, c.Download(fg))
	assertNoRemoteActions(t)
	assert.Zero(t, server.calls.BlobReads.Load(), "should not download anything again")

	outDir := fg.OutDir()
	assertFileContent(t, filepath.Join(outDir, "fg_download_dep_out.txt"), dep.File)
	assertFileContent(t, filepath.Join(outDir, "fg_download_dep_dir/sub/nested.txt"), dep.Nested)
	src, err := os.ReadFile("package/src1.txt")
	require.NoError(t, err)
	assertFileContent(t, filepath.Join(outDir, "src1.txt"), src)
}

func TestFilegroupDownloadDownloadsDependencies(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	build.Init(c.state)
	dep := buildRemoteDep(t, c, core.BuildLabel{PackageName: "package2", Name: "fg_download_deps_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_download_deps"}, dep.Target.Label)
	_, err := c.Build(fg)
	require.NoError(t, err)

	require.NoError(t, c.Download(fg))
	assertFileContent(t, filepath.Join(fg.OutDir(), "fg_download_deps_dep_out.txt"), dep.File)
	assertFileContent(t, filepath.Join(fg.OutDir(), "fg_download_deps_dep_dir/sub/nested.txt"), dep.Nested)
}

func TestSamePackageFilegroupDownloadKeepsDependencyUpToDate(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	build.Init(c.state)
	dep := buildRemoteDep(t, c, core.BuildLabel{PackageName: "package3", Name: "fg_same_pkg_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package3", Name: "fg_same_pkg"}, dep.Target.Label)
	_, err := c.Build(fg)
	require.NoError(t, err)

	require.NoError(t, c.Download(dep.Target))
	require.NoError(t, c.Download(fg))
	// The filegroup's outputs are the same files as its dependency's; downloading it must not
	// make the dependency look out of date (which would cause it to be downloaded again next time).
	assert.True(t, c.outputsExist(dep.Target, c.unstampedBuildActionDigests.Get(dep.Target.Label)))
	assertFileContent(t, filepath.Join(dep.Target.OutDir(), "fg_same_pkg_dep_out.txt"), dep.File)
}

// newRemoteSubrepoFilegroup builds a remote subrepo whose build_defs package contains the given directory,
// and returns a filegroup in that package with the given sources.
func newRemoteSubrepoFilegroup(t *testing.T, c *Client, name string, pkgDir *pb.Directory, children []*pb.Directory, srcs ...string) *core.BuildTarget {
	t.Helper()
	root := &pb.Directory{Directories: []*pb.DirectoryNode{{Name: "build_defs", Digest: putProto(pkgDir)}}}
	putProto(root)
	server.mockActionResult = &pb.ActionResult{
		OutputDirectories: []*pb.OutputDirectory{{
			Path:       name + "_root",
			TreeDigest: putProto(&pb.Tree{Root: root, Children: append([]*pb.Directory{pkgDir}, children...)}),
		}},
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			QueuedTimestamp:             timestamppb.Now(),
			ExecutionStartTimestamp:     timestamppb.Now(),
			ExecutionCompletedTimestamp: timestamppb.Now(),
		},
	}
	subrepoTarget := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: name})
	subrepoTarget.AddOutput(name + "_root")
	subrepoTarget.IsSubrepo = true
	subrepoTarget.Command = "echo " + name
	c.state.Graph.AddTarget(subrepoTarget)
	_, err := c.Build(subrepoTarget)
	require.NoError(t, err)
	subrepoTarget.SetState(core.BuiltRemotely)

	subrepo := core.NewSubrepo(c.state, name, name+"_root", subrepoTarget, c.state.Arch, false)
	require.True(t, subrepo.IsRemoteSubrepo())
	fg := core.NewBuildTarget(core.BuildLabel{Subrepo: name, PackageName: "build_defs", Name: "defs"})
	fg.IsFilegroup = true
	fg.Subrepo = subrepo
	for _, src := range srcs {
		fg.AddSource(core.SubrepoFileLabel{File: src, Package: "build_defs", FullPackage: subrepo.Dir("build_defs")})
	}
	c.state.Graph.AddTarget(fg)
	_, err = c.Build(fg)
	require.NoError(t, err)
	return fg
}

func TestRemoteSubrepoFilegroupDownload(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	c.state.RemoteClient = c
	build.Init(c.state)

	// The subrepo contains two files; the filegroup only uses one of them.
	defs := []byte("def some_rule():\n    pass\n")
	pkgDir := &pb.Directory{Files: []*pb.FileNode{
		{Name: "defs.build_defs", Digest: putBlob(defs)},
		{Name: "other.txt", Digest: putBlob([]byte("some other file in the subrepo"))},
	}}
	server.calls.Reset()
	fg := newRemoteSubrepoFilegroup(t, c, "fg_subrepo", pkgDir, nil, "defs.build_defs")
	server.calls.Reset()

	require.NoError(t, c.Download(fg))
	assertNoRemoteActions(t)
	assertFileContent(t, filepath.Join(fg.OutDir(), "defs.build_defs"), defs)
	// We only need the one file from the subrepo, so shouldn't have downloaded the rest of it.
	assert.EqualValues(t, 1, server.calls.BlobReads.Load())
}

func TestRemoteSubrepoFilegroupDownloadUpdatesHash(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	c.state.RemoteClient = c
	build.Init(c.state)
	defs := []byte("new content\n")
	defsDigest := putBlob(defs)
	pkgDir := &pb.Directory{Files: []*pb.FileNode{{Name: "defs.build_defs", Digest: defsDigest}}}
	fg := newRemoteSubrepoFilegroup(t, c, "fg_subrepo_hash", pkgDir, nil, "defs.build_defs")

	// An old version of the file exists from a previous build, and has been hashed already.
	out := filepath.Join(fg.OutDir(), "defs.build_defs")
	require.NoError(t, os.MkdirAll(filepath.Dir(out), 0755))
	require.NoError(t, os.WriteFile(out, []byte("old content\n"), 0644))
	_, err := c.state.PathHasher.Hash(out, false, true, false)
	require.NoError(t, err)

	require.NoError(t, c.Download(fg))
	assertFileContent(t, out, defs)
	h, err := c.state.PathHasher.Hash(out, false, true, false)
	require.NoError(t, err)
	assert.Equal(t, defsDigest.Hash, hex.EncodeToString(h), "should not have a stale hash for the downloaded file")
}

func TestRemoteSubrepoFilegroupDownloadSkipsUpToDateOutputs(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	c.state.RemoteClient = c
	build.Init(c.state)
	defs := []byte("some build defs")
	nested := []byte("a file in a directory")
	dataDir := &pb.Directory{Files: []*pb.FileNode{{Name: "nested.txt", Digest: putBlob(nested)}}}
	pkgDir := &pb.Directory{
		Files:       []*pb.FileNode{{Name: "defs.build_defs", Digest: putBlob(defs)}},
		Directories: []*pb.DirectoryNode{{Name: "data", Digest: putProto(dataDir)}},
		Symlinks:    []*pb.SymlinkNode{{Name: "link.build_defs", Target: "defs.build_defs"}},
	}
	fg := newRemoteSubrepoFilegroup(t, c, "fg_subrepo_skip", pkgDir, []*pb.Directory{dataDir}, "defs.build_defs", "data", "link.build_defs")

	require.NoError(t, c.downloadFilegroup(fg))
	assertFileContent(t, filepath.Join(fg.OutDir(), "defs.build_defs"), defs)
	assertFileContent(t, filepath.Join(fg.OutDir(), "data/nested.txt"), nested)
	assertFileContent(t, filepath.Join(fg.OutDir(), "link.build_defs"), defs)

	// Downloading again (e.g. in a later build) shouldn't fetch anything.
	server.calls.Reset()
	require.NoError(t, c.downloadFilegroup(fg))
	assert.Zero(t, server.calls.BlobReads.Load(), "should not download anything again")
	assertFileContent(t, filepath.Join(fg.OutDir(), "data/nested.txt"), nested)
}

// buildRemoteNamedDep builds a target remotely with two named outputs, a and b.
func buildRemoteNamedDep(t *testing.T, c *Client, label core.BuildLabel) (dep *core.BuildTarget, a, b []byte) {
	t.Helper()
	a = []byte("named output a of " + label.String())
	b = []byte("named output b of " + label.String())
	server.mockActionResult = &pb.ActionResult{
		OutputFiles: []*pb.OutputFile{
			{Path: label.Name + "_a.txt", Digest: putBlob(a)},
			{Path: label.Name + "_b.txt", Digest: putBlob(b)},
		},
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			QueuedTimestamp:             timestamppb.Now(),
			ExecutionStartTimestamp:     timestamppb.Now(),
			ExecutionCompletedTimestamp: timestamppb.Now(),
		},
	}
	dep = core.NewBuildTarget(label)
	dep.AddNamedOutput("a", label.Name+"_a.txt")
	dep.AddNamedOutput("b", label.Name+"_b.txt")
	dep.Command = "echo " + label.String()
	c.state.Graph.AddTarget(dep)
	_, err := c.Build(dep)
	require.NoError(t, err)
	dep.SetState(core.BuiltRemotely)
	return dep, a, b
}

func TestFilegroupNamedOutputDownloadOnlyFetchesNamedOutput(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	build.Init(c.state)
	dep, a, _ := buildRemoteNamedDep(t, c, core.BuildLabel{PackageName: "package2", Name: "fg_named_dl_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_named_dl"},
		core.AnnotatedOutputLabel{BuildLabel: dep.Label, Annotation: "a"})
	_, err := c.Build(fg)
	require.NoError(t, err)

	server.calls.Reset()
	require.NoError(t, c.Download(fg))
	assertNoRemoteActions(t)
	assertFileContent(t, filepath.Join(fg.OutDir(), "fg_named_dl_dep_a.txt"), a)
	assert.EqualValues(t, 1, server.calls.BlobReads.Load(), "should only download the named output")
	assert.NoFileExists(t, filepath.Join(dep.OutDir(), "fg_named_dl_dep_b.txt"), "should not download the whole dependency")
}

func TestFilegroupNamedOutputDownloadReusesDownloadedDependency(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")
	build.Init(c.state)
	dep, a, _ := buildRemoteNamedDep(t, c, core.BuildLabel{PackageName: "package2", Name: "fg_named_reuse_dep"})
	fg := newFilegroup(c, core.BuildLabel{PackageName: "package", Name: "fg_named_reuse"},
		core.AnnotatedOutputLabel{BuildLabel: dep.Label, Annotation: "a"})
	_, err := c.Build(fg)
	require.NoError(t, err)
	require.NoError(t, c.Download(dep))

	server.calls.Reset()
	require.NoError(t, c.Download(fg))
	assert.Zero(t, server.calls.BlobReads.Load(), "the dependency is already downloaded, so shouldn't fetch anything")
	assertFileContent(t, filepath.Join(fg.OutDir(), "fg_named_reuse_dep_a.txt"), a)
}

func assertFileContent(t *testing.T, filename string, expected []byte) {
	t.Helper()
	b, err := os.ReadFile(filename)
	if assert.NoError(t, err) {
		assert.Equal(t, expected, b)
	}
}
