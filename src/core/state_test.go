package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var a = []LineCoverage{NotExecutable, Uncovered, Uncovered, Covered, NotExecutable, Unreachable}
var b = []LineCoverage{Uncovered, Covered, Uncovered, Uncovered, Unreachable, Covered}
var c = []LineCoverage{Covered, NotExecutable, Covered, Uncovered, Covered, NotExecutable}
var empty = []LineCoverage{}

func TestMergeCoverageLines1(t *testing.T) {
	coverage := MergeCoverageLines(a, b)
	expected := []LineCoverage{Uncovered, Covered, Uncovered, Covered, Unreachable, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines2(t *testing.T) {
	coverage := MergeCoverageLines(a, c)
	expected := []LineCoverage{Covered, Uncovered, Covered, Covered, Covered, Unreachable}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines3(t *testing.T) {
	coverage := MergeCoverageLines(b, c)
	expected := []LineCoverage{Covered, Covered, Covered, Uncovered, Covered, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines4(t *testing.T) {
	coverage := MergeCoverageLines(MergeCoverageLines(a, b), c)
	expected := []LineCoverage{Covered, Covered, Covered, Covered, Covered, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines5(t *testing.T) {
	coverage := MergeCoverageLines(MergeCoverageLines(c, a), b)
	expected := []LineCoverage{Covered, Covered, Covered, Covered, Covered, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines6(t *testing.T) {
	coverage := MergeCoverageLines(empty, b)
	assert.Equal(t, b, coverage)
}

func TestMergeCoverageLines7(t *testing.T) {
	coverage := MergeCoverageLines(a, empty)
	assert.Equal(t, a, coverage)
}

func TestMergeCoverageLines8(t *testing.T) {
	coverage := MergeCoverageLines(empty, empty)
	assert.Equal(t, empty, coverage)
}

func TestExpandOriginalTargets(t *testing.T) {
	state := NewBuildState(1, nil, 4, DefaultConfiguration())
	state.OriginalTargets = []BuildLabel{{"src/core", "all"}, {"src/parse", "parse"}}
	state.Include = []string{"go"}
	state.Exclude = []string{"py"}

	addTarget(state, "//src/core:target1", "go")
	addTarget(state, "//src/core:target2", "py")
	addTarget(state, "//src/core:target3", "go", "py")
	addTarget(state, "//src/core:target4")
	addTarget(state, "//src/core:target5", "go", "manual")
	addTarget(state, "//src/parse:parse")
	addTarget(state, "//src/parse:parse2", "go")

	// Only two targets come out here; target2 has 'py' so is excluded, target3 has
	// both 'py' and 'go' but excludes take priority, target4 doesn't have 'go',
	// and target5 has 'go' but 'manual' should also take priority.
	// //src/parse:parse doesn't have 'go' but was explicitly requested so will be
	// added anyway.
	assert.Equal(t, state.ExpandOriginalTargets(), BuildLabels{
		{"src/core", "target1"},
		{"src/parse", "parse"},
	})
}

func TestExpandOriginalTestTargets(t *testing.T) {
	state := NewBuildState(1, nil, 4, DefaultConfiguration())
	state.OriginalTargets = []BuildLabel{{"src/core", "all"}}
	state.NeedTests = true
	state.Include = []string{"go"}
	state.Exclude = []string{"py"}
	addTarget(state, "//src/core:target1", "go")
	addTarget(state, "//src/core:target2", "py")
	addTarget(state, "//src/core:target1_test", "go")
	addTarget(state, "//src/core:target2_test", "py")
	addTarget(state, "//src/core:target3_test")
	addTarget(state, "//src/core:target4_test", "go", "manual")
	// Only the one target comes out here; it must be a test and otherwise follows
	// the same include / exclude logic as the previous test.
	assert.Equal(t, state.ExpandOriginalTargets(), BuildLabels{{"src/core", "target1_test"}})
}

func TestComparePendingTasks(t *testing.T) {
	p := func(taskType TaskType) pendingTask { return pendingTask{Type: taskType} }
	// NB. "Higher priority" means the task comes first, does not refer to numeric values.
	assertHigherPriority := func(a, b TaskType) {
		// relationship should be commutative
		assert.True(t, p(a).Compare(p(b)) < 0)
		assert.True(t, p(b).Compare(p(a)) > 0)
	}
	assertEqualPriority := func(a, b TaskType) {
		assert.True(t, p(a).Compare(p(b)) == 0)
		assert.True(t, p(b).Compare(p(a)) == 0)
	}

	assertHigherPriority(SubincludeBuild, SubincludeParse)
	assertHigherPriority(SubincludeParse, Build)
	assertHigherPriority(SubincludeBuild, Build)
	assertEqualPriority(Build, Parse)
	assertEqualPriority(Build, Test)
	assertEqualPriority(Parse, Test)
	assertHigherPriority(Build, Stop)
	assertHigherPriority(Test, Stop)
	assertHigherPriority(Parse, Stop)

	// sanity check
	assertEqualPriority(SubincludeBuild, SubincludeBuild)
	assertEqualPriority(SubincludeParse, SubincludeParse)
	assertEqualPriority(Build, Build)
	assertEqualPriority(Parse, Parse)
	assertEqualPriority(Test, Test)
	assertEqualPriority(Stop, Stop)
}

func addTarget(state *BuildState, name string, labels ...string) {
	target := NewBuildTarget(ParseBuildLabel(name, ""))
	target.Labels = labels
	target.IsTest = strings.HasSuffix(name, "_test")
	pkg := state.Graph.Package(target.Label.PackageName)
	if pkg == nil {
		pkg = NewPackage(target.Label.PackageName)
		state.Graph.AddPackage(pkg)
	}
	pkg.Targets[target.Label.Name] = target
	state.Graph.AddTarget(target)
}
