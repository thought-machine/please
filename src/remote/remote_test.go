package remote

import (
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestInit(t *testing.T) {
	c := newClient()
	assert.NoError(t, c.CheckInitialised())
}

func TestBadAPIVersion(t *testing.T) {
	// We specify a required API version of v2.0.0, so should fail initialisation if the server
	// specifies something incompatible with that.
	defer server.Reset()
	server.HighApiVersion.Major = 1
	server.LowApiVersion.Major = 1
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

func TestStoreAndRetrieve(t *testing.T) {
	c := newClient()
	c.CheckInitialised()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target1"})
	target.AddSource(core.FileLabel{File: "src1.txt", Package: "package"})
	target.AddSource(core.FileLabel{File: "src2.txt", Package: "package"})
	target.AddOutput("out1.txt")
	target.PostBuildFunction = testFunction{}
	metadata := &core.BuildMetadata{
		Stdout:    []byte("test stdout"),
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
	}
	err := c.Store(target, metadata, []string{"out1.txt"})
	assert.NoError(t, err)
	// Remove the old file, but remember its contents so we can compare later.
	contents, err := ioutil.ReadFile("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	err = os.Remove("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	// Now retrieve back the output of this thing.
	retrievedMetadata, err := c.Retrieve(target)
	assert.NoError(t, err)
	cachedContents, err := ioutil.ReadFile("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	assert.Equal(t, contents, cachedContents)
	assert.Equal(t, metadata, retrievedMetadata)
}

func TestStoreAndRetrieveDir(t *testing.T) {
	c := newClient()
	c.CheckInitialised()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package2", Name: "target2"})
	target.IsBinary = true
	target.IsTest = true
	target.SetState(core.Built)
	target.AddLabel(core.TestResultsDirLabel)
	target.AddOutput("target2")
	c.state.Graph.AddTarget(target)
	err := c.Store(target, &core.BuildMetadata{}, target.Outputs())
	assert.NoError(t, err)
	err = c.Store(target, &core.BuildMetadata{
		Stdout: []byte("test stdout"),
		Test:   true,
	}, []string{
		path.Base(target.TestResultsFile()),
		"1.xml",
	})
	assert.NoError(t, err)

	// Move old results out of the way
	resultsDir := "plz-out/bin/package2/.test_results_target2"
	err = os.RemoveAll(resultsDir)
	assert.NoError(t, err)
	err = os.Remove("plz-out/bin/package2/1.xml")
	assert.NoError(t, err)

	// Now retrieve them again
	_, err = c.Retrieve(target)
	assert.NoError(t, err)
	b, err := ioutil.ReadFile(path.Join(resultsDir, "1.xml"))
	assert.NoError(t, err)
	assert.Equal(t, string(b), xmlResults)
	b, err = ioutil.ReadFile(path.Join(resultsDir, "results/2.xml"))
	assert.NoError(t, err)
	assert.Equal(t, string(b), xmlResults)

	// TODO(peterebden): This does not seem to be working on OSX or FreeBSD; it doesn't seem
	//                   to be because of this package, but the test files are seemingly not
	//                   coming out as symlinks correctly.
	if runtime.GOOS == "linux" {
		link, err := os.Readlink("plz-out/bin/package2/1.xml")
		assert.NoError(t, err)
		assert.Equal(t, ".test_results_target2/1.xml", link)
		link, err = os.Readlink(path.Join(resultsDir, "2.xml"))
		assert.NoError(t, err)
		assert.Equal(t, "results/2.xml", link)
	}
}

const xmlResults = `<?xml version="1.0" encoding="UTF-8"?>
<testsuite errors="0" failures="0" name="src.build.python.pex_test.PexTest-20150416153858" tests="1" time="0.000">
<properties/>
<testcase classname="src.build.python.pex_test.PexTest" name="testSuccess" time="0.000"/>           <testcase classname="src.build.python.pex_test.PexTest" name="testSuccess" time="0.000">
        </testcase>
</testsuite>
`

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

type postBuildFunction func(*core.BuildTarget, string) error

func (f postBuildFunction) Call(target *core.BuildTarget, output string) error {
	return f(target, output)
}
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

func TestExecuteTest(t *testing.T) {
	c := newClientInstance("test")
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target3"})
	target.AddOutput("remote_test")
	target.TestTimeout = time.Minute
	target.TestCommand = "$TEST"
	target.IsTest = true
	target.IsBinary = true
	target.SetState(core.Building)
	err := c.Store(target, &core.BuildMetadata{}, target.Outputs())
	assert.NoError(t, err)
	c.state.Graph.AddTarget(target)
	_, results, coverage, err := c.Test(0, target)
	assert.NoError(t, err)
	assert.Equal(t, testResults, results)
	assert.Equal(t, 0, len(coverage)) // Wasn't requested
}

func TestExecuteTestWithCoverage(t *testing.T) {
	c := newClientInstance("test")
	c.state.NeedCoverage = true // bit of a hack but we need to turn this on somehow
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target4"})
	target.AddOutput("remote_test")
	target.TestTimeout = time.Minute
	target.TestCommand = "$TEST"
	target.IsTest = true
	target.IsBinary = true
	err := c.Store(target, &core.BuildMetadata{}, target.Outputs())
	assert.NoError(t, err)
	target.SetState(core.Built)
	c.state.Graph.AddTarget(target)
	_, results, coverage, err := c.Test(0, target)
	assert.NoError(t, err)
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
	cmd, _ := c.buildCommand(target, &pb.Directory{}, false)
	testDir := os.Getenv("TEST_DIR")
	for _, env := range cmd.EnvironmentVariables {
		assert.False(t, path.IsAbs(env.Value), "Env var %s has an absolute path: %s", env.Name, env.Value)
		assert.NotContains(t, env.Value, core.OutDir, "Env var %s contains %s: %s", env.Name, core.OutDir, env.Value)
		assert.NotContains(t, env.Value, testDir, "Env var %s contains the test dir %s: %s", env.Name, testDir, env.Value)
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
	cmd, _ := c.buildCommand(target, &pb.Directory{}, false)
	for _, env := range cmd.EnvironmentVariables {
		assert.False(t, path.IsAbs(env.Value), "Env var %s has an absolute path: %s", env.Name, env.Value)
		assert.NotContains(t, env.Value, core.OutDir, "Env var %s contains %s: %s", env.Name, core.OutDir, env.Value)
	}
}
