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
	"reflect"
	"runtime/debug"
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
	t.Skip("Still working on getting these to work reliably")
	// Test modifying a command in the pre-build function.
	state, target := newState("//package1:target6")
	target.AddOutput("file6")
	target.Command = "" // Target now won't produce the needed output
	f := func() error {
		target.Command = "echo 'wibble wibble wibble' > $OUT"
		return nil
	}
	target.PreBuildFunction = reflect.ValueOf(&f).Pointer()
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPostBuildFunction(t *testing.T) {
	t.Skip("Still working on getting these to work reliably")
	// Test modifying a command in the post-build function.
	state, target := newState("//package1:target7")
	target.Command = "echo 'wibble wibble wibble' | tee file7"
	f := func(s string) error {
		target.AddOutput("file7")
		assert.Equal(t, "wibble wibble wibble", s)
		return nil
	}
	target.PostBuildFunction = reflect.ValueOf(&f).Pointer()
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
	state.Cache = &cache
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPostBuildFunctionAndCache(t *testing.T) {
	t.Skip("Still working on getting these to work reliably")
	// Test the often subtle and quick to anger interaction of post-build function and cache.
	// In this case when it fails to retrieve the post-build output it should still call the function after building.
	state, target := newState("//package1:target9")
	target.AddOutput("file9")
	target.Command = "echo 'wibble wibble wibble' | tee $OUT"
	called := false
	f := func(s string) error {
		called = true
		assert.Equal(t, "wibble wibble wibble", s)
		return nil
	}
	target.PostBuildFunction = reflect.ValueOf(&f).Pointer()
	state.Cache = &cache
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
	assert.True(t, called)
}

func TestPostBuildFunctionAndCache2(t *testing.T) {
	t.Skip("Still working on getting these to work reliably")
	// Test the often subtle and quick to anger interaction of post-build function and cache.
	// In this case it succeeds in retrieving the post-build output but must still call the function.
	state, target := newState("//package1:target10")
	target.AddOutput("file10")
	target.Command = "echo 'wibble wibble wibble' | tee $OUT"
	called := false
	f := func(s string) error {
		assert.False(t, called, "Must only call post-build function once (issue #113)")
		called = true
		assert.Equal(t, "retrieved from cache", s) // comes from implementation below
		return nil
	}
	target.PostBuildFunction = reflect.ValueOf(&f).Pointer()
	state.Cache = &cache
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
	assert.True(t, called)
}

func newState(label string) (*core.BuildState, *core.BuildTarget) {
	config, _ := core.ReadConfigFiles(nil)
	state := core.NewBuildState(1, nil, 4, config)
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.Command = fmt.Sprintf("echo 'output of %s' > $OUT", target.Label)
	state.Graph.AddTarget(target)
	return state, target
}

// Fake cache implementation with hardcoded behaviour for the various tests above.
type mockCache struct{}

func (*mockCache) Store(target *core.BuildTarget, key []byte) {
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
	if target.Label.Name == "target10" && file == core.PostBuildOutputFileName(target) {
		ioutil.WriteFile(postBuildOutputFileName(target), []byte("retrieved from cache"), 0664)
		return true
	}
	return false
}

func (*mockCache) Clean(target *core.BuildTarget) {
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
	// This is not at all nice but it keeps the GC from collecting the
	// pre- and post-build functions in tests before they get called.
	debug.SetGCPercent(-1)
	os.Exit(m.Run())
}
