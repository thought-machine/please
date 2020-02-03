// Tests around the main part of the build process.
// These are somewhat fiddly because by its nature the code has many side effects.
// We attempt to minimise some through mocking.
//
// Note that because the tests run in an indeterminate order and maybe in parallel
// they all have to be careful to use distinct build targets.

package build

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var cache core.Cache

func TestBuildTargetWithNoDeps(t *testing.T) {
	state, target := newState("//package1:target1")
	target.AddOutput("file1")
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestFailedBuildTarget(t *testing.T) {
	state, target := newState("//package1:target1a")
	target.Command = "false"
	err := buildTarget(1, state, target, false)
	assert.Error(t, err)
}

func TestBuildTargetWhichNeedsRebuilding(t *testing.T) {
	// The output file for this target already exists, but it should still get rebuilt
	// because there's no rule hash file.
	state, target := newState("//package1:target2")
	target.AddOutput("file2")
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestBuildTargetWhichDoesntNeedRebuilding(t *testing.T) {
	// We write a rule hash for this target before building it, so we don't need to build again.
	state, target := newState("//package1:target3")
	target.AddOutput("file3")
	assert.NoError(t, writeRuleHash(state, target))
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Reused, target.State())
}

func TestModifiedBuildTargetStillNeedsRebuilding(t *testing.T) {
	// Similar to above, but if we change the target such that the rule hash no longer matches,
	// it should get rebuilt.
	state, target := newState("//package1:target4")
	target.AddOutput("file4")
	assert.NoError(t, writeRuleHash(state, target))
	target.Command = "echo 'wibble wibble wibble' > $OUT"
	target.RuleHash = nil // Have to force a reset of this
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestSymlinkedOutputs(t *testing.T) {
	// Test behaviour when the output is a symlink.
	state, target := newState("//package1:target5")
	target.AddOutput("file5")
	target.AddSource(core.FileLabel{File: "src5", Package: "package1"})
	target.Command = "ln -s $SRC $OUT"
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPreBuildFunction(t *testing.T) {
	// Test modifying a command in the pre-build function.
	state, target := newState("//package1:target6")
	target.AddOutput("file6")
	target.Command = "" // Target now won't produce the needed output
	target.PreBuildFunction = preBuildFunction(func(target *core.BuildTarget) error {
		target.Command = "echo 'wibble wibble wibble' > $OUT"
		return nil
	})
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPostBuildFunction(t *testing.T) {
	// Test modifying a command in the post-build function.
	state, target := newState("//package1:target7")
	target.Command = "echo 'wibble wibble wibble' | tee file7"
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		target.AddOutput("file7")
		assert.Equal(t, "wibble wibble wibble", output)
		return nil
	})
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
	assert.Equal(t, []string{"file7"}, target.Outputs())
}

func TestCacheRetrieval(t *testing.T) {
	// Test retrieving stuff from the cache
	state, target := newState("//package1:target8")
	target.AddOutput("file8")
	target.Command = "false" // Will fail if we try to build it.
	state.Cache = cache
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Cached, target.State())
}

func TestPostBuildFunctionAndCache(t *testing.T) {
	// Test the often subtle and quick to anger interaction of post-build function and cache.
	// In this case when it fails to retrieve the post-build output it should still call the function after building.
	state, target := newState("//package1:target9")
	target.AddOutput("file9")
	target.Command = "echo 'wibble wibble wibble' | tee $OUT"
	called := false
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		called = true
		assert.Equal(t, "wibble wibble wibble", output)
		return nil
	})
	state.Cache = cache
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
	assert.True(t, called)
}

func TestPostBuildFunctionAndCache2(t *testing.T) {
	// Test the often subtle and quick to anger interaction of post-build function and cache.
	// In this case it succeeds in retrieving the post-build output but must still call the function.
	state, target := newState("//package1:target10")
	target.AddOutput("file10")
	target.Command = "echo 'wibble wibble wibble' | tee $OUT"
	called := false
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		assert.False(t, called, "Must only call post-build function once (issue #113)")
		called = true
		assert.Equal(t, "retrieved from cache", output) // comes from implementation below
		return nil
	})
	state.Cache = cache
	err := buildTarget(1, state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Cached, target.State())
	assert.True(t, called)
}

func TestInitPyCreation(t *testing.T) {
	state, _ := newState("//pypkg:wevs")
	target1 := newPyFilegroup(state, "//pypkg:target1", "file1.py")
	target2 := newPyFilegroup(state, "//pypkg:target2", "__init__.py")
	_, err := buildFilegroup(state, target1)
	assert.NoError(t, err)
	assert.True(t, fs.FileExists("plz-out/gen/pypkg/__init__.py"))
	_, err = buildFilegroup(state, target2)
	assert.NoError(t, err)
	d, err := ioutil.ReadFile("plz-out/gen/pypkg/__init__.py")
	assert.NoError(t, err)
	assert.Equal(t, `"""output from //pypkg:target2"""`, strings.TrimSpace(string(d)))
}

func TestRecursiveInitPyCreation(t *testing.T) {
	state, _ := newState("//package1/package2:wevs")
	target1 := newPyFilegroup(state, "//package1/package2:target1", "file1.py")
	_, err := buildFilegroup(state, target1)
	assert.NoError(t, err)
	assert.True(t, fs.FileExists("plz-out/gen/package1/package2/__init__.py"))
	assert.True(t, fs.FileExists("plz-out/gen/package1/__init__.py"))
}

func TestCreatePlzOutGo(t *testing.T) {
	state, target := newState("//gopkg:target")
	target.AddLabel("link:plz-out/go/${PKG}/src")
	target.AddOutput("file1.go")
	assert.False(t, fs.PathExists("plz-out/go"))
	assert.NoError(t, buildTarget(1, state, target, false))
	assert.True(t, fs.PathExists("plz-out/go/gopkg/src/file1.go"))
}

func TestLicenceEnforcement(t *testing.T) {
	state, target := newState("//pkg:good")
	state.Config.Licences.Reject = append(state.Config.Licences.Reject, "gpl")
	state.Config.Licences.Accept = append(state.Config.Licences.Accept, "mit")

	// Target specifying no licence should not panic.
	checkLicences(state, target)

	// A license (non case sensitive) that is not in the list of accepted licenses will panic.
	assert.Panics(t, func() {
		target.Licences = append(target.Licences, "Bsd")
		checkLicences(state, target)
	}, "A target with a non-accepted licence will panic")

	// Accepting bsd should resolve the panic
	state.Config.Licences.Accept = append(state.Config.Licences.Accept, "BSD")
	checkLicences(state, target)

	// Now construct a new "bad" target.
	state, target = newState("//pkg:bad")
	state.Config.Licences.Reject = append(state.Config.Licences.Reject, "gpl")
	state.Config.Licences.Accept = append(state.Config.Licences.Accept, "mit")

	// Adding an explicitly rejected licence should panic no matter what.
	target.Licences = append(target.Licences, "GPL")
	assert.Panics(t, func() {
		checkLicences(state, target)
	}, "Trying to add GPL should panic (case insensitive)")
}

func TestFileGroupBinDir(t *testing.T) {
	state, target := newState("//package1:bindir")
	//target.AddOutput("test_data")
	target.AddSource(core.FileLabel{File: "package2", Package: target.Label.PackageName})
	target.IsBinary = true
	target.IsFilegroup = true

	_, err := buildFilegroup(state, target)
	assert.NoError(t, err)

	assert.True(t, fs.PathExists("plz-out/bin/package1/package2/"))
	assert.True(t, fs.FileExists("plz-out/bin/package1/package2/file1.py"))
	assert.True(t, fs.IsDirectory("plz-out/bin/package1/package2/"))

	// Ensure correct permission on directory
	info, err := os.Stat("plz-out/bin/package1/package2/")
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestOutputHash(t *testing.T) {
	state, target := newState("//package3:target1")
	target.AddOutput("file1")
	target.Hashes = []string{"6c6d66a0852b49cdeeb0e183b4f10b0309c5dd4a"}
	b, err := state.TargetHasher.OutputHash(target)
	assert.NoError(t, err)
	assert.Equal(t, "6c6d66a0852b49cdeeb0e183b4f10b0309c5dd4a", hex.EncodeToString(b))
}

func TestCheckRuleHashes(t *testing.T) {
	state, target := newState("//package3:target1")
	target.AddOutput("file1")
	target.Hashes = []string{"6c6d66a0852b49cdeeb0e183b4f10b0309c5dd4a"}

	// This is the normal sha1-with-combine hash calculation
	b, _ := state.TargetHasher.OutputHash(target)
	err := checkRuleHashes(state, target, b)
	assert.NoError(t, err)

	// This is testing the negative case
	target.Hashes = []string{"630bff40cc8d5329e6176779493281ddb3e0add3"}
	err = checkRuleHashes(state, target, b)
	assert.Error(t, err)

	// This is the equivalent to sha1sum of the file, so should be accepted too
	target.Hashes = []string{"dba7673010f19a94af4345453005933fd511bea9"}
	err = checkRuleHashes(state, target, b)
	assert.NoError(t, err)

	// This is the equivalent to sha256sum of the file, so should be accepted too
	target.Hashes = []string{"634b027b1b69e1242d40d53e312b3b4ac7710f55be81f289b549446ef6778bee"}
	err = checkRuleHashes(state, target, b)
	assert.NoError(t, err)
}

func newState(label string) (*core.BuildState, *core.BuildTarget) {
	config, _ := core.ReadConfigFiles(nil, nil)
	state := core.NewBuildState(config)
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.Command = fmt.Sprintf("echo 'output of %s' > $OUT", target.Label)
	target.BuildTimeout = 100 * time.Second
	state.Graph.AddTarget(target)
	state.Parser = &fakeParser{}
	Init(state)
	return state, target
}

func newPyFilegroup(state *core.BuildState, label, filename string) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.AddSource(core.FileLabel{File: filename, Package: target.Label.PackageName})
	target.AddOutput(filename)
	target.AddLabel("py")
	target.IsFilegroup = true
	state.Graph.AddTarget(target)
	return target
}

// Fake cache implementation with hardcoded behaviour for the various tests above.
type mockCache struct{}

func (*mockCache) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
}

func (*mockCache) Retrieve(target *core.BuildTarget, key []byte, outputs []string) *core.BuildMetadata {
	if target.Label.Name == "target8" {
		ioutil.WriteFile("plz-out/gen/package1/file8", []byte("retrieved from cache"), 0664)
		return &core.BuildMetadata{}
	} else if target.Label.Name == "target10" {
		ioutil.WriteFile("plz-out/gen/package1/file10", []byte("retrieved from cache"), 0664)
		return &core.BuildMetadata{Stdout: []byte("retrieved from cache")}
	}
	return nil
}

func (*mockCache) Clean(target *core.BuildTarget) {}
func (*mockCache) CleanAll()                      {}
func (*mockCache) Shutdown()                      {}

type fakeParser struct {
}

func (fake *fakeParser) ParseFile(state *core.BuildState, pkg *core.Package, filename string) error {
	return nil
}

func (fake *fakeParser) ParseReader(state *core.BuildState, pkg *core.Package, r io.ReadSeeker) error {
	return nil
}

func (fake *fakeParser) RunPreBuildFunction(threadID int, state *core.BuildState, target *core.BuildTarget) error {
	return target.PreBuildFunction.Call(target)
}

func (fake *fakeParser) RunPostBuildFunction(threadID int, state *core.BuildState, target *core.BuildTarget, output string) error {
	return target.PostBuildFunction.Call(target, output)
}

type preBuildFunction func(*core.BuildTarget) error
type postBuildFunction func(*core.BuildTarget, string) error

func (f preBuildFunction) Call(target *core.BuildTarget) error { return f(target) }
func (f preBuildFunction) String() string                      { return "" }
func (f postBuildFunction) Call(target *core.BuildTarget, output string) error {
	return f(target, output)
}
func (f postBuildFunction) String() string { return "" }

func TestMain(m *testing.M) {
	cache = &mockCache{}
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.DEBUG, "")
	logging.SetBackend(backend, backendLeveled)
	// Move ourselves to the root of the test data tree
	wd, _ := os.Getwd()
	core.RepoRoot = path.Join(wd, "src/build/test_data")
	Init(core.NewDefaultBuildState())
	if err := os.Chdir(core.RepoRoot); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
