package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
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
	metadata, err := c.Build(target)
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
	_, err := c.Build(target)
	assert.NoError(t, err)
	assert.Equal(t, []string{"somefile"}, target.Outputs(c.state.Graph))
}

func TestExecuteFetch(t *testing.T) {
	c := newClient()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "remote1"})
	target.IsRemoteFile = true
	target.AddSource(core.URLLabel("https://get.please.build/linux_amd64/14.2.0/please_14.2.0.tar.gz"))
	target.AddOutput("please_14.2.0.tar.gz")
	target.Hashes = []string{"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}
	target.BuildTimeout = time.Minute
	_, err := c.Build(target)
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
	_, err = c.Test(target, 1)
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
	_, err = c.Test(target, 1)
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
	cmd, _ := c.buildCommand(target, &pb.Directory{}, false, false, false, 0)
	testDir := os.Getenv("TEST_DIR")
	for _, env := range cmd.EnvironmentVariables {
		if !strings.HasPrefix(env.Value, "//") {
			assert.False(t, filepath.IsAbs(env.Value), "Env var %s has an absolute path: %s", env.Name, env.Value)
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
	cmd, _ := c.buildCommand(target, &pb.Directory{}, false, false, false, 0)
	for _, env := range cmd.EnvironmentVariables {
		if !strings.HasPrefix(env.Value, "//") {
			assert.False(t, filepath.IsAbs(env.Value), "Env var %s has an absolute path: %s", env.Name, env.Value)
			assert.NotContains(t, env.Value, core.OutDir, "Env var %s contains %s: %s", env.Name, core.OutDir, env.Value)
		}
	}
}

func TestRemoteFilesHashConsistently(t *testing.T) {
	c := newClientInstance("test")
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "download"})
	target.IsRemoteFile = true
	target.AddSource(core.URLLabel("https://localhost/file"))
	cmd, digest, err := c.buildAction(target, false, false, 0)
	assert.NoError(t, err)
	// After we change this path, the rule should still give back the same protos since it is
	// not relevant to how we fetch a remote asset.
	c.state.Config.Build.Path = []string{"/usr/bin/nope"}
	cmd2, digest2, err := c.buildAction(target, false, false, 0)
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

	c.state.AddOriginalTarget(outDirTarget.Label)
	c.state.OutputDownload = core.OriginalOutputDownload
	require.True(t, c.state.ShouldDownload(outDirTarget))

	outDirTarget.AddOutputDirectory("foo")
	// Doesn't actually get executed but gives an idea as to how this rule is mocked up
	outDirTarget.Command = "touch foo/bar.txt && touch foo/baz.txt"
	c.state.Graph.AddTarget(outDirTarget)
	_, err := c.Build(outDirTarget)
	require.NoError(t, err)

	assert.Len(t, outDirTarget.Outputs(c.state.Graph), 2)
	assert.ElementsMatch(t, []string{"foo.txt", "bar.txt"}, outDirTarget.Outputs(c.state.Graph))
	for _, out := range outDirTarget.Outputs(c.state.Graph) {
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
	cmd, err := c.buildCommand(target, &pb.Directory{}, false, false, false, 0)
	assert.NoError(t, err)
	assert.Equal(t, &pb.Platform{
		Properties: []*pb.Platform_Property{
			{
				Name:  "OSFamily",
				Value: "linux",
			},
		},
	}, cmd.Platform) //nolint:staticcheck

	target.Labels = []string{"remote-platform-property:size=chomky"}
	cmd, err = c.buildCommand(target, &pb.Directory{}, false, false, false, 0)
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
	}, cmd.Platform) //nolint:staticcheck
}

// Store is a small hack that stores a target's outputs for testing only.
func (c *Client) Store(target *core.BuildTarget) error {
	if err := c.CheckInitialised(); err != nil {
		return err
	}
	return c.uploadLocalTarget(target)
}

func TestBuildTestCommand(t *testing.T) {
	c := newClientInstance("test")
	state := c.state
	state.TestArgs = []string{"--foo", "--bar"}

	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target_placeholder"})
	target.AddOutput("remote_test")
	target.Test = &core.TestFields{
		Timeout:         time.Minute,
		Command:         "$TEST __TEST_ARGS__ 2>&1",
		ArgsPlaceholder: "__TEST_ARGS__",
	}
	target.IsBinary = true

	cmd, err := c.buildTestCommand(state, target, 1)
	assert.NoError(t, err)

	assert.True(t,
		strings.HasSuffix(cmd.Arguments[len(cmd.Arguments)-1], "$TEST --foo --bar 2>&1"),
		`expected suffix "$TEST --foo --bar 2>&1" on %q`, cmd.Arguments[len(cmd.Arguments)-1],
	)
}

func TestNegotiateAPIVersion(t *testing.T) {
	v := func(major, minor int32) *semver.SemVer {
		return &semver.SemVer{Major: major, Minor: minor}
	}
	tests := []struct {
		name                    string
		deprecated, low, high   *semver.SemVer
		expected                *semver.SemVer
		expectDeprecated, fails bool
	}{
		{name: "please-servers mettle", low: v(2, 0), high: v(2, 1), expected: v(2, 1)},
		{name: "please-servers flair", low: v(2, 0), high: v(2, 3), expected: v(2, 3)},
		{name: "Buildbarn", deprecated: v(2, 2), low: v(2, 3), high: v(2, 12), expected: v(2, 12)},
		{name: "newer than us", low: v(2, 3), high: v(2, 20), expected: v(2, 12)},
		{name: "only deprecated overlap", deprecated: v(2, 0), low: v(2, 13), high: v(2, 14), expected: v(2, 12), expectDeprecated: true},
		{name: "too old", low: v(2, 0), high: v(2, 0), fails: true},
		{name: "too new", deprecated: v(2, 13), low: v(2, 14), high: v(3, 0), fails: true},
		{name: "unset", fails: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version, deprecated, err := negotiateAPIVersion(&pb.ServerCapabilities{
				DeprecatedApiVersion: test.deprecated,
				LowApiVersion:        test.low,
				HighApiVersion:       test.high,
			})
			if test.fails {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, printVer(test.expected), printVer(version))
			assert.Equal(t, test.expectDeprecated, deprecated)
		})
	}
}

func TestOutputSymlinks(t *testing.T) {
	link := &pb.OutputSymlink{Path: "link", Target: "file"}
	dirLink := &pb.OutputSymlink{Path: "dirlink", Target: "dir"}
	// A 2.1+ server only populates output_symlinks
	assert.Equal(t, []*pb.OutputSymlink{link, dirLink}, outputSymlinks(&pb.ActionResult{
		OutputSymlinks: []*pb.OutputSymlink{link, dirLink},
	}))
	// One compatible with 2.0 populates both, which shouldn't be duplicated
	assert.Equal(t, []*pb.OutputSymlink{link, dirLink}, outputSymlinks(&pb.ActionResult{
		OutputSymlinks:          []*pb.OutputSymlink{link, dirLink},
		OutputFileSymlinks:      []*pb.OutputSymlink{link},
		OutputDirectorySymlinks: []*pb.OutputSymlink{dirLink},
	}))
	// A 2.0 server only populates the deprecated fields
	assert.Equal(t, []*pb.OutputSymlink{link, dirLink}, outputSymlinks(&pb.ActionResult{
		OutputFileSymlinks:      []*pb.OutputSymlink{link},
		OutputDirectorySymlinks: []*pb.OutputSymlink{dirLink},
	}))
}

func TestSDKActionResult(t *testing.T) {
	link := &pb.OutputSymlink{Path: "link", Target: "file"}
	ar := &pb.ActionResult{OutputSymlinks: []*pb.OutputSymlink{link}}
	sdkAR := sdkActionResult(ar)
	assert.Equal(t, []*pb.OutputSymlink{link}, sdkAR.OutputFileSymlinks)                    //nolint:staticcheck
	assert.Empty(t, ar.OutputFileSymlinks, "original action result should not be modified") //nolint:staticcheck
	// If the deprecated fields are already populated, it should be left alone.
	ar = &pb.ActionResult{OutputSymlinks: []*pb.OutputSymlink{link}, OutputFileSymlinks: []*pb.OutputSymlink{link}}
	assert.Same(t, ar, sdkActionResult(ar))
}

// TestOutputSymlinksDownloaded builds a target against a server that only populates output_symlinks
// (as Buildbarn does) and checks that we record and download the symlink.
func TestOutputSymlinksDownloaded(t *testing.T) {
	defer server.Reset()
	c := newClientInstance("mock")

	content := []byte("this is the content of the real file")
	contentDigest := digest.NewFromBlob(content)
	server.blobs[contentDigest.Hash] = content
	server.mockActionResult = &pb.ActionResult{
		OutputFiles:    []*pb.OutputFile{{Path: "real.txt", Digest: contentDigest.ToProto()}},
		OutputSymlinks: []*pb.OutputSymlink{{Path: "link.txt", Target: "real.txt"}},
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			Worker:                      "kev",
			QueuedTimestamp:             timestamppb.Now(),
			ExecutionStartTimestamp:     timestamppb.Now(),
			ExecutionCompletedTimestamp: timestamppb.Now(),
		},
	}

	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "symlink_target"})
	target.AddOutput("real.txt")
	target.AddOutput("link.txt")
	target.Command = "echo hello > real.txt && ln -s real.txt link.txt"
	c.state.AddOriginalTarget(target.Label)
	c.state.OutputDownload = core.OriginalOutputDownload
	c.state.Graph.AddTarget(target)
	_, err := c.Build(target)
	require.NoError(t, err)

	outs := c.targetOutputs(target.Label)
	require.NotNil(t, outs)
	require.Len(t, outs.Symlinks, 1)
	assert.Equal(t, "link.txt", outs.Symlinks[0].Name)
	assert.Equal(t, "real.txt", outs.Symlinks[0].Target)

	dest, err := os.Readlink(filepath.Join(target.OutDir(), "link.txt"))
	require.NoError(t, err)
	assert.Equal(t, "real.txt", dest)
}
