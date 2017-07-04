// Tests around the main part of the build process.
// These are somewhat fiddly because by its nature the code has many side effects.
// We attempt to minimise some through mocking.
//
// Note that because the tests run in an indeterminate order and maybe in parallel
// they all have to be careful to use distinct build targets.

package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/op/go-logging.v1"

	"core"
)

var cache core.Cache

func TestBuildTargetWithNoDeps(t *testing.T) {
	state, target := newState("//package1:target1")
	target.AddOutput("file1")
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestFailedBuildTarget(t *testing.T) {
	state, target := newState("//package1:target1a")
	target.Command = "false"
	err := buildTarget(1, state, target)
	assert.Error(t, err)
}

func TestBuildTargetWhichNeedsRebuilding(t *testing.T) {
	// The output file for this target already exists, but it should still get rebuilt
	// because there's no rule hash file.
	state, target := newState("//package1:target2")
	target.AddOutput("file2")
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestBuildTargetWhichDoesntNeedRebuilding(t *testing.T) {
	// We write a rule hash file for this target before building it, so we don't need to build again.
	state, target := newState("//package1:target3")
	target.AddOutput("file3")
	assert.NoError(t, writeRuleHashFile(state, target))
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Reused, target.State())
}

func TestModifiedBuildTargetStillNeedsRebuilding(t *testing.T) {
	// Similar to above, but if we change the target such that the rule hash no longer matches,
	// it should get rebuilt.
	state, target := newState("//package1:target4")
	target.AddOutput("file4")
	assert.NoError(t, writeRuleHashFile(state, target))
	target.Command = "echo 'wibble wibble wibble' > $OUT"
	target.RuleHash = nil // Have to force a reset of this
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestSymlinkedOutputs(t *testing.T) {
	// Test behaviour when the output is a symlink.
	state, target := newState("//package1:target5")
	target.AddOutput("file5")
	target.AddSource(core.FileLabel{File: "src5", Package: "package1"})
	target.Command = "ln -s $SRC $OUT"
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	// Unchanged because input and output are the same.
	assert.Equal(t, core.Unchanged, target.State())
}

func TestPreBuildFunction(t *testing.T) {
	// Test modifying a command in the pre-build function.
	state, target := newState("//package1:target6")
	target.AddOutput("file6")
	target.Command = ""                // Target now won't produce the needed output
	target.PreBuildFunction = 12345678 // Needs to be nonzero so build knows to do something with it.
	state.Parser.(*fakeParser).PreBuildFunctions[target] = func(target *core.BuildTarget, output string) error {
		target.Command = "echo 'wibble wibble wibble' > $OUT"
		return nil
	}
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPostBuildFunction(t *testing.T) {
	// Test modifying a command in the post-build function.
	state, target := newState("//package1:target7")
	target.Command = "echo 'wibble wibble wibble' | tee file7"
	target.PostBuildFunction = 12345678 // Again, needs to be nonzero.
	state.Parser.(*fakeParser).PostBuildFunctions[target] = func(target *core.BuildTarget, output string) error {
		target.AddOutput("file7")
		assert.Equal(t, "wibble wibble wibble", output)
		return nil
	}
	err := buildTarget(1, state, target)
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
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Cached, target.State())
}

func TestPostBuildFunctionAndCache(t *testing.T) {
	// Test the often subtle and quick to anger interaction of post-build function and cache.
	// In this case when it fails to retrieve the post-build output it should still call the function after building.
	state, target := newState("//package1:target9")
	target.AddOutput("file9")
	target.Command = "echo 'wibble wibble wibble' | tee $OUT"
	target.PostBuildFunction = 12345678
	called := false
	state.Parser.(*fakeParser).PostBuildFunctions[target] = func(target *core.BuildTarget, output string) error {
		called = true
		assert.Equal(t, "wibble wibble wibble", output)
		return nil
	}
	state.Cache = cache
	err := buildTarget(1, state, target)
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
	target.PostBuildFunction = 12345678
	called := false
	state.Parser.(*fakeParser).PostBuildFunctions[target] = func(target *core.BuildTarget, output string) error {
		assert.False(t, called, "Must only call post-build function once (issue #113)")
		called = true
		assert.Equal(t, "retrieved from cache", output) // comes from implementation below
		return nil
	}
	state.Cache = cache
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Cached, target.State())
	assert.True(t, called)
}

func TestInitPyCreation(t *testing.T) {
	state, _ := newState("//pypkg:wevs")
	target1 := newPyFilegroup(state, "//pypkg:target1", "file1.py")
	target2 := newPyFilegroup(state, "//pypkg:target2", "__init__.py")
	assert.NoError(t, buildFilegroup(0, state, target1))
	assert.True(t, core.FileExists("plz-out/gen/pypkg/__init__.py"))
	assert.NoError(t, buildFilegroup(0, state, target2))
	d, err := ioutil.ReadFile("plz-out/gen/pypkg/__init__.py")
	assert.NoError(t, err)
	assert.Equal(t, `"""output from //pypkg:target2"""`, strings.TrimSpace(string(d)))
}

func TestRecursiveInitPyCreation(t *testing.T) {
	state, _ := newState("//package1/package2:wevs")
	target1 := newPyFilegroup(state, "//package1/package2:target1", "file1.py")
	assert.NoError(t, buildFilegroup(0, state, target1))
	assert.True(t, core.FileExists("plz-out/gen/package1/package2/__init__.py"))
	assert.True(t, core.FileExists("plz-out/gen/package1/__init__.py"))
}

func TestCreatePlzOutGo(t *testing.T) {
	state, target := newState("//gopkg:target")
	target.AddLabel("go")
	target.AddOutput("file1.go")
	assert.False(t, core.PathExists("plz-out/go"))
	assert.NoError(t, buildTarget(1, state, target))
	assert.True(t, core.PathExists("plz-out/go/src"))
	assert.True(t, core.PathExists("plz-out/go/pkg/"+runtime.GOOS+"_"+runtime.GOARCH))
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

func newState(label string) (*core.BuildState, *core.BuildTarget) {
	config, _ := core.ReadConfigFiles(nil)
	state := core.NewBuildState(1, nil, 4, config)
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.Command = fmt.Sprintf("echo 'output of %s' > $OUT", target.Label)
	state.Graph.AddTarget(target)
	state.Parser = &fakeParser{
		PostBuildFunctions: buildFunctionMap{},
		PreBuildFunctions:  buildFunctionMap{},
	}
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

func (*mockCache) Store(target *core.BuildTarget, key []byte, files ...string) {
}

func (*mockCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
}

func (*mockCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	if target.Label.Name == "target8" {
		ioutil.WriteFile("plz-out/gen/package1/file8", []byte("retrieved from cache"), 0664)
		return true
	} else if target.Label.Name == "target10" {
		ioutil.WriteFile("plz-out/gen/package1/file10", []byte("retrieved from cache"), 0664)
		return true
	}
	return false
}

func (*mockCache) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	if target.Label.Name == "target10" && file == target.PostBuildOutputFileName() {
		ioutil.WriteFile(postBuildOutputFileName(target), []byte("retrieved from cache"), 0664)
		return true
	}
	return false
}

func (*mockCache) Clean(target *core.BuildTarget) {}
func (*mockCache) CleanAll()                      {}
func (*mockCache) Shutdown()                      {}

type buildFunctionMap map[*core.BuildTarget]func(*core.BuildTarget, string) error

type fakeParser struct {
	PostBuildFunctions buildFunctionMap
	PreBuildFunctions  buildFunctionMap
}

func (fake *fakeParser) RunPreBuildFunction(threadId int, state *core.BuildState, target *core.BuildTarget) error {
	if f, present := fake.PreBuildFunctions[target]; present {
		return f(target, "")
	}
	return nil
}

func (fake *fakeParser) RunPostBuildFunction(threadId int, state *core.BuildState, target *core.BuildTarget, output string) error {
	if f, present := fake.PostBuildFunctions[target]; present {
		return f(target, output)
	}
	return nil
}

func (fake *fakeParser) UndeferAnyParses(state *core.BuildState, target *core.BuildTarget) {
}

func TestMain(m *testing.M) {
	cache = &mockCache{}
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.DEBUG, "")
	logging.SetBackend(backend, backendLeveled)
	// Move ourselves to the root of the test data tree
	wd, _ := os.Getwd()
	core.RepoRoot = path.Join(wd, "src/build/test_data")
	if err := os.Chdir(core.RepoRoot); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
