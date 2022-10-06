package remote

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

func TestInit(t *testing.T) {
	c := newClient()
	assert.NoError(t, c.CheckInitialised())
}

func TestBadAPIVersion(t *testing.T) {
	// We specify a required API version of v2.0.0, so should fail initialisation if the server
	// specifies something incompatible with that.
	defer server.Reset()
	server.HighAPIVersion.Major = 1
	server.LowAPIVersion.Major = 1
	c := newClient()
	assert.Error(t, c.CheckInitialised())
	assert.Contains(t, c.CheckInitialised().Error(), "1.0.0 - 1.1.0")
}

func TestUnsupportedDigest(t *testing.T) {
	defer server.Reset()
	server.DigestFunction = []pb.DigestFunction_Value{
		pb.DigestFunction_MD5,
		pb.DigestFunction_SHA384,
		pb.DigestFunction_SHA512,
	}
	c := newClient()
	assert.Error(t, c.CheckInitialised())
}

func TestExecuteBuild(t *testing.T) {
	c := newClient()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target2"})
	target.AddSource(core.FileLabel{File: "src1.txt", Package: "package"})
	target.AddSource(core.FileLabel{File: "src2.txt", Package: "package"})
	target.AddOutput("out2.txt")
	target.BuildTimeout = time.Minute
	// We need to set this to force stdout to be retrieved (it is otherwise unnecessary
	// on success).
	target.PostBuildFunction = testFunction{}
	target.Command = "echo hello && echo test > $OUT"
	metadata, err := c.Build(0, target)
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello\n"), metadata.Stdout)
}

type postBuildFunction func(*core.BuildTarget, string) error //nolint:unused

//nolint:unused
func (f postBuildFunction) Call(target *core.BuildTarget, output string) error {
	return f(target, output)
}

//nolint:unused
func (f postBuildFunction) String() string { return "" }

func TestExecutePostBuildFunction(t *testing.T) {
	t.Skip("Post-build function currently triggered at a higher level")
	c := newClient()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target5"})
	target.BuildTimeout = time.Minute
	target.Command = "echo 'wibble wibble wibble' | tee file7"
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		target.AddOutput("somefile")
		assert.Equal(t, "wibble wibble wibble", output)
		return nil
	})
	_, err := c.Build(0, target)
	assert.NoError(t, err)
	assert.Equal(t, []string{"somefile"}, target.Outputs())
}

func TestExecuteFetch(t *testing.T) {
	c := newClient()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "remote1"})
	target.IsRemoteFile = true
	target.AddSource(core.URLLabel("https://get.please.build/linux_amd64/14.2.0/please_14.2.0.tar.gz"))
	target.AddOutput("please_14.2.0.tar.gz")
	target.Hashes = []string{"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}
	target.BuildTimeout = time.Minute
	_, err := c.Build(0, target)
	assert.NoError(t, err)
}

func TestExecuteTest(t *testing.T) {
	c := newClientInstance("test")
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target3"})
	target.AddOutput("remote_test")
	target.Test = new(core.TestFields)
	target.Test.Timeout = time.Minute
	target.Test.Command = "$TEST"
	target.IsBinary = true
	target.SetState(core.Building)
	err := c.Store(target)
	assert.NoError(t, err)
	c.state.Graph.AddTarget(target)
	_, err = c.Test(0, target, 1)
	assert.NoError(t, err)

	results, err := os.ReadFile(filepath.Join(target.TestDir(1), core.TestResultsFile))
	require.NoError(t, err)

	assert.Equal(t, testResults, results)
}

func TestExecuteTestWithCoverage(t *testing.T) {
	c := newClientInstance("test")
	c.state.NeedCoverage = true // bit of a hack but we need to turn this on somehow
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target4"})
	target.AddOutput("remote_test")
	target.Test = new(core.TestFields)
	target.Test.Timeout = time.Minute
	target.Test.Command = "$TEST"
	target.IsBinary = true
	err := c.Store(target)
	assert.NoError(t, err)
	target.SetState(core.Built)
	c.state.Graph.AddTarget(target)
	_, err = c.Test(0, target, 1)
	assert.NoError(t, err)

	results, err := os.ReadFile(filepath.Join(target.TestDir(1), core.TestResultsFile))
	require.NoError(t, err)

	coverage, err := os.ReadFile(filepath.Join(target.TestDir(1), core.CoverageFile))
	require.NoError(t, err)

	assert.Equal(t, testResults, results)
	assert.Equal(t, coverageData, coverage)
}

var testResults = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<testcase name="//src/remote:remote_test">
  <test name="testResults" success="true" time="172" type="SUCCESS"/>
</testcase>
`)

var coverageData = []byte(`mode: set
src/core/build_target.go:134.54,143.2 7 0
src/core/build_target.go:159.52,172.2 12 0
src/core/build_target.go:177.44,179.2 1 0
`)

func TestNoAbsolutePaths(t *testing.T) {
	c := newClientInstance("test")
	tool := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "tool"})
	tool.AddOutput("bin")
	c.state.Graph.AddTarget(tool)
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target5"})
	target.AddOutput("remote_test")
	target.AddSource(core.FileLabel{Package: "package", File: "file"})
	target.AddTool(tool.Label)
	cmd, _ := c.buildCommand(target, &pb.Directory{}, false, false, false)
	testDir := os.Getenv("TEST_DIR")
	for _, env := range cmd.EnvironmentVariables {
		if !strings.HasPrefix(env.Value, "//") {
			assert.False(t, path.IsAbs(env.Value), "Env var %s has an absolute path: %s", env.Name, env.Value)
			assert.NotContains(t, env.Value, core.OutDir, "Env var %s contains %s: %s", env.Name, core.OutDir, env.Value)
			assert.NotContains(t, env.Value, testDir, "Env var %s contains the test dir %s: %s", env.Name, testDir, env.Value)
		}
	}
}

func TestNoAbsolutePaths2(t *testing.T) {
	c := newClientInstance("test")
	tool := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "tool"})
	tool.AddOutput("bin")
	c.state.Graph.AddTarget(tool)
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target5"})
	target.AddOutput("remote_test")
	target.AddTool(core.SystemPathLabel{Path: []string{os.Getenv("TMP_DIR")}, Name: "remote_test"})
	cmd, _ := c.buildCommand(target, &pb.Directory{}, false, false, false)
	for _, env := range cmd.EnvironmentVariables {
		if !strings.HasPrefix(env.Value, "//") {
			assert.False(t, path.IsAbs(env.Value), "Env var %s has an absolute path: %s", env.Name, env.Value)
			assert.NotContains(t, env.Value, core.OutDir, "Env var %s contains %s: %s", env.Name, core.OutDir, env.Value)
		}
	}
}

func TestRemoteFilesHashConsistently(t *testing.T) {
	c := newClientInstance("test")
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "download"})
	target.IsRemoteFile = true
	target.AddSource(core.URLLabel("https://localhost/file"))
	cmd, digest, err := c.buildAction(target, false, false)
	assert.NoError(t, err)
	// After we change this path, the rule should still give back the same protos since it is
	// not relevant to how we fetch a remote asset.
	c.state.Config.Build.Path = []string{"/usr/bin/nope"}
	cmd2, digest2, err := c.buildAction(target, false, false)
	assert.NoError(t, err)
	assert.Equal(t, cmd, cmd2)
	assert.Equal(t, digest, digest2)
}

func TestOutDirsSetOutsOnTarget(t *testing.T) {
	c := newClientInstance("mock")

	foo := []byte("this is the content of foo")
	fooDigest := digest.NewFromBlob(foo)

	bar := []byte("this is the content of bar")
	barDigest := digest.NewFromBlob(bar)

	tree := mustMarshal(&pb.Tree{
		Root: &pb.Directory{
			Files: []*pb.FileNode{
				{Name: "foo.txt", Digest: fooDigest.ToProto()},
				{Name: "bar.txt", Digest: barDigest.ToProto()},
			},
		},
	})
	treeDigest := digest.NewFromBlob(tree)

	server.mockActionResult = &pb.ActionResult{
		OutputDirectories: []*pb.OutputDirectory{
			{
				Path:       "foo",
				TreeDigest: treeDigest.ToProto(),
			},
		},
		ExitCode: 0,
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			Worker:                      "kev",
			QueuedTimestamp:             timestamppb.Now(),
			ExecutionStartTimestamp:     timestamppb.Now(),
			ExecutionCompletedTimestamp: timestamppb.Now(),
		},
	}

	server.blobs[treeDigest.Hash] = tree

	server.blobs[fooDigest.Hash] = foo
	server.blobs[barDigest.Hash] = bar

	outDirTarget := core.NewBuildTarget(core.BuildLabel{
		PackageName: "package",
		Name:        "out_dir_target",
	})

	c.state.AddOriginalTarget(outDirTarget.Label, true)
	c.state.OutputDownload = core.OriginalOutputDownload
	require.True(t, c.state.ShouldDownload(outDirTarget))

	outDirTarget.AddOutputDirectory("foo")
	// Doesn't actually get executed but gives an idea as to how this rule is mocked up
	outDirTarget.Command = "touch foo/bar.txt && touch foo/baz.txt"
	c.state.Graph.AddTarget(outDirTarget)
	_, err := c.Build(0, outDirTarget)
	require.NoError(t, err)

	assert.Len(t, outDirTarget.Outputs(), 2)
	assert.ElementsMatch(t, []string{"foo.txt", "bar.txt"}, outDirTarget.Outputs())
	for _, out := range outDirTarget.Outputs() {
		assert.True(t, fs.FileExists(filepath.Join(outDirTarget.OutDir(), out)), "output %s doesn't exist in target out folder", out)
	}
}

func TestDirectoryMetadataStore(t *testing.T) {
	cacheDuration := time.Hour
	now := time.Now().UTC()

	store := directoryMetadataStore{
		directory:     storeDirectoryName,
		cacheDuration: cacheDuration,
	}

	mds := map[string]*core.BuildMetadata{
		"delete": {
			Timestamp: now.Add(-cacheDuration * 2),
		},
		"keep": {
			Timestamp: now,
		},
	}

	for key, value := range mds {
		err := store.storeMetadata(key, value)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(storeDirectoryName, key[:2], key))
	}

	md, err := store.retrieveMetadata("delete")
	require.NoError(t, err)
	assert.Nil(t, md)

	md, err = store.retrieveMetadata("keep")
	require.NoError(t, err)
	assert.Equal(t, md, mds["keep"])

	store.clean()

	assert.FileExists(t, filepath.Join(storeDirectoryName, "ke", "keep"))

	_, err = os.Lstat(filepath.Join(storeDirectoryName, "de", "delete"))
	assert.True(t, os.IsNotExist(err))
}

func TestTargetPlatform(t *testing.T) {
	c := newClientInstance("platform_test")
	c.platform = convertPlatform(c.state.Config.Remote.Platform) // Bit of a hack but we can't go through the normal path.
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target"})
	cmd, err := c.buildCommand(target, &pb.Directory{}, false, false, false)
	assert.NoError(t, err)
	assert.Equal(t, &pb.Platform{
		Properties: []*pb.Platform_Property{
			{
				Name:  "OSFamily",
				Value: "linux",
			},
		},
	}, cmd.Platform)

	target.Labels = []string{"remote-platform-property:size=chomky"}
	cmd, err = c.buildCommand(target, &pb.Directory{}, false, false, false)
	assert.NoError(t, err)
	assert.Equal(t, &pb.Platform{
		Properties: []*pb.Platform_Property{
			{
				Name:  "size",
				Value: "chomky",
			},
			{
				Name:  "OSFamily",
				Value: "linux",
			},
		},
	}, cmd.Platform)
}

// Store is a small hack that stores a target's outputs for testing only.
func (c *Client) Store(target *core.BuildTarget) error {
	if err := c.CheckInitialised(); err != nil {
		return err
	}
	return c.uploadLocalTarget(target)
}
