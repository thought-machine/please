package parse

import (
	"os"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"core"
)

func TestParseSourceBuildLabel(t *testing.T) {
	src, err := parseSource("//src/parse/test_data/test_subfolder4:test_py", "src/parse", false)
	assert.NoError(t, err)
	label := src.Label()
	assert.NotNil(t, label)
	assert.Equal(t, label.PackageName, "src/parse/test_data/test_subfolder4")
	assert.Equal(t, label.Name, "test_py")
}

func TestParseSourceRelativeBuildLabel(t *testing.T) {
	src, err := parseSource(":builtin_rules", "src/parse", false)
	assert.NoError(t, err)
	label := src.Label()
	assert.NotNil(t, label)
	assert.Equal(t, "src/parse", label.PackageName)
	assert.Equal(t, "builtin_rules", label.Name)
}

// Test parsing from a subdirectory that does not contain a build file.
func TestParseSourceFromSubdirectory(t *testing.T) {
	src, err := parseSource("test_subfolder3/test_py", "src/parse/test_data", false)
	assert.NoError(t, err)
	assert.Nil(t, src.Label())
	paths := src.Paths(nil)
	assert.Equal(t, 1, len(paths))
	assert.Equal(t, "src/parse/test_data/test_subfolder3/test_py", paths[0])
}

func TestParseSourceFromOwnedSubdirectory(t *testing.T) {
	_, err := parseSource("test_subfolder4/test_py", "src/parse/test_data", false)
	assert.Error(t, err, "Should produce an error when parsing from a subdirectory that does contain a build file")
}

func TestParseSourceWithParentPath(t *testing.T) {
	_, err := parseSource("test_subfolder4/../test_py", "src/parse/test_data", false)
	assert.Error(t, err, "Should produce an error when parsing a path with ../ in it")
}

func TestParseSourceWithAbsolutePath(t *testing.T) {
	_, err := parseSource("/test_subfolder4/test_py", "src/parse/test_data", false)
	assert.Error(t, err, "Should produce an error trying to parse an absolute path")
	_, err = parseSource("/usr/bin/go", "src/parse/test_data", true)
	assert.NoError(t, err, "Should not produce an error trying to parse an absolute path in cases where it's allowed")
}

func TestAddTarget(t *testing.T) {
	pkg := core.NewPackage("src/parse")
	addTargetTest1 := func(name string, binary, container, test bool, testCmd string) *core.BuildTarget {
		return addTarget(uintptr(unsafe.Pointer(pkg)), name, "true", testCmd, binary, test,
			false, false, container, false, false, false, 0, 0, 0, "Building...")
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

	assert.Panics(t, func() { addTargetTest(":target1", false, false) },
		"Should return nil attempting to add a target with an illegal name")
	assert.Nil(t, addTargetTest("target1", false, false),
		"Should return nil attempting to add a new target with the same name")

	assert.Nil(t, core.State.Graph.Target(core.ParseBuildLabel("//src/parse:target1", "")),
		"Shouldn't have added target to the graph yet")
	core.State.Graph.AddPackage(pkg)
	addTargetTest("target6", true, false)
	target6 := core.State.Graph.Target(core.ParseBuildLabel("//src/parse:target6", ""))
	assert.NotNil(t, target6, "Should have been added to the graph since the package is added")
	assert.True(t, target6.HasLabel("bin"))
}

func TestGetSubincludeFile(t *testing.T) {
	assertError := func(t *testing.T, ret, msg string) { assert.True(t, strings.HasPrefix(ret, "__"), msg) }

	state := core.NewBuildState(10, nil, 2, core.DefaultConfiguration())
	pkg := core.NewPackage("src/parse")
	pkg2 := core.NewPackage("src/core")
	assert.Equal(t, pyDeferParse, getSubincludeFile(pkg, "//src/core:target"), "Package not loaded yet, should defer")
	assertError(t, getSubincludeFile(pkg, "//src/parse:target"), "Should produce an error on attempts for local subincludes.")
	assertError(t, getSubincludeFile(pkg, ":target"), "Should produce an error on attempts for local subincludes.")
	state.Graph.AddPackage(pkg)
	state.Graph.AddPackage(pkg2)
	assertError(t, getSubincludeFile(pkg, "//src/core:target"), "Produces an error, target does not exist in package.")
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/core:target", ""))
	state.Graph.AddTarget(target)
	assertError(t, getSubincludeFile(pkg, "//src/core:target"), "Errors, target is not visible to subincluding package.")
	target.Visibility = []core.BuildLabel{core.ParseBuildLabel("//src/parse:all", "")}
	assertError(t, getSubincludeFile(pkg, "//src/core:target"), "Errors, target doesn't have any outputs to include.")
	target.AddOutput("test.py")
	assert.Equal(t, pyDeferParse, getSubincludeFile(pkg, "//src/core:target"), "Target isn't built yet, so still deferred")
	target.SetState(core.Built)
	assert.Equal(t, "plz-out/gen/src/core/test.py", getSubincludeFile(pkg, "//src/core:target"), "Success at last")
}

func TestMain(m *testing.M) {
	core.NewBuildState(10, nil, 2, core.DefaultConfiguration())
	// Need to set this before calling parseSource.
	core.State.Config.Please.BuildFileName = []string{"TEST_BUILD"}
	os.Exit(m.Run())
}
