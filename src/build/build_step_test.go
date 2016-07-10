// Tests around the main part of the build process.
// These are somewhat fiddly because by its nature the code has many side effects.
// We attempt to minimise some through mocking.
//
// Note that because the tests run in an indeterminate order and maybe in parallel
// they all have to be careful to use distinct build targets.

package build

import (
	"fmt"
	"os"
	"path"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"gopkg.in/op/go-logging.v1"

	"core"
)

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
	target.Command = "" // Target now won't produce the needed output
	f := func() error {
		target.Command = "echo 'wibble wibble wibble' > $OUT"
		return nil
	}
	target.PreBuildFunction = uintptr(unsafe.Pointer(&f))
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPostBuildFunction(t *testing.T) {
	// Test modifying a command in the post-build function.
	state, target := newState("//package1:target7")
	target.Command = "echo 'wibble wibble wibble' > file7"
	f := func() error {
		target.AddOutput("file7")
		return nil
	}
	target.PreBuildFunction = uintptr(unsafe.Pointer(&f))
	err := buildTarget(1, state, target)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
	assert.Equal(t, []string{"file7"}, target.Outputs())
}

func TestPostBuildFunctionAndCache(t *testing.T) {

}

func newState(label string) (*core.BuildState, *core.BuildTarget) {
	config, _ := core.ReadConfigFiles(nil)
	state := core.NewBuildState(1, nil, 4, config)
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.Command = fmt.Sprintf("echo 'output of %s' > $OUT", target.Label)
	state.Graph.AddTarget(target)
	return state, target
}

func TestMain(m *testing.M) {
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
