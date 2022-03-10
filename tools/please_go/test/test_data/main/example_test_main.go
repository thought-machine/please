// This isn't a 'real' source file, it's test data for //src/build/go:write_test_main_test
// Code generated for the purpose of the above test. DO NOT EDIT.

package parse

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestParseSourceBuildLabel(t *testing.T) {
	src := parseSource("//src/parse/test_data/test_subfolder4:test_py", "src/parse")
	label, ok := src.Label()
	assert.True(t, ok)
	assert.Equal(t, label.PackageName, "src/parse/test_data/test_subfolder4")
	assert.Equal(t, label.Name, "test_py")
}

func TestParseSourceRelativeBuildLabel(t *testing.T) {
	src := parseSource(":builtin_rules", "src/parse")
	label, ok := src.Label()
	assert.True(t, ok)
	assert.Equal(t, "src/parse", label.PackageName)
	assert.Equal(t, "builtin_rules", label.Name)
}

// Test parsing from a subdirectory that does not contain a build file.
func TestParseSourceFromSubdirectory(t *testing.T) {
	src := parseSource("test_subfolder3/test_py", "src/parse/test_data")
	_, ok := src.Label()
	assert.False(t, ok)
	paths := src.Paths(nil)
	assert.Equal(t, 1, len(paths))
	assert.Equal(t, "src/parse/test_data/test_subfolder3/test_py", paths[0])
}

func TestParseSourceFromOwnedSubdirectory(t *testing.T) {
	assert.Panics(t, func() { parseSource("test_subfolder4/test_py", "src/parse/test_data") },
		"Should panic when parsing from a subdirectory that does contain a build file")
}

func TestParseSourceWithParentPath(t *testing.T) {
	assert.Panics(t, func() { parseSource("test_subfolder4/../test_py", "src/parse/test_data") },
		"Should panic when parsing a path with ../ in it")
}

func TestParseSourceWithAbsolutePath(t *testing.T) {
	assert.Panics(t, func() { parseSource("/test_subfolder4/test_py", "src/parse/test_data") },
		"Should panic trying to parse an absolute path")
}

func TestAddTarget(t *testing.T) {
	state := core.NewDefaultBuildState()
	pkg := core.NewPackage("src/parse")
	addTargetTest1 := func(name string, binary, container, test bool, testCmd string) *core.BuildTarget {
		return &core.BuildTarget{}
	}
	addTargetTest := func(name string, binary, container bool) *core.BuildTarget {
		return addTargetTest1(name, binary, container, false, "")
	}
	// Test that labels are correctly applied
	target1 := addTargetTest("target1", false, false)
	assert.False(t, target1.HasLabel("bin"))
	assert.False(t, target1.HasLabel("container"))
	target2 := addTargetTest("target2", true, false)
	assert.True(t, target2.HasLabel("bin"))
	assert.False(t, target2.HasLabel("container"))
	target3 := addTargetTest("target3", true, true)
	assert.True(t, target3.HasLabel("bin"))
	assert.True(t, target3.HasLabel("container"))

	assert.Panics(t, func() { addTargetTest("target1", false, false) },
		"Should panic attempting to add a new target with the same name")
	assert.Panics(t, func() { addTargetTest1("target4", false, false, true, "") },
		"Should panic attempting to add a test target with no test command")
	assert.Panics(t, func() { addTargetTest1("target5", false, false, false, "true") },
		"Should panic attempting to add a non-test target with a test command")

	assert.Nil(t, state.Graph.Target(core.ParseBuildLabel("//src/parse:target1", "")),
		"Shouldn't have added target to the graph yet")
	state.Graph.AddPackage(pkg)
	addTargetTest("target6", true, false)
	target6 := state.Graph.Target(core.ParseBuildLabel("//src/parse:target6", ""))
	assert.NotNil(t, target6, "Should have been added to the graph since the package is added")
	assert.True(t, target6.HasLabel("bin"))
}

func TestMain(m *testing.M) {
	core.NewDefaultBuildState()
	os.Exit(m.Run())
}

func parseSource(src, dir string) core.BuildLabel {
	panic("nope")
}
