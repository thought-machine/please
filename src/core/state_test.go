package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandOriginalLabels(t *testing.T) {
	state := NewDefaultBuildState()
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core", Name: "all"}, true)
	state.AddOriginalTarget(BuildLabel{PackageName: "src/parse", Name: "parse"}, true)
	state.Include = []string{"go"}
	state.Exclude = []string{"py"}

	addTarget(state, "//src/core:target1", "go")
	addTarget(state, "//src/core:target2", "py")
	addTarget(state, "//src/core:target3", "go", "py")
	addTarget(state, "//src/core:target4")
	addTarget(state, "//src/parse:parse")
	addTarget(state, "//src/parse:parse2", "go")

	// Only two targets come out here; target2 has 'py' so is excluded, target3 has
	// both 'py' and 'go' but excludes take priority, target4 doesn't have 'go',
	// and target5 has 'go' but 'manual' should also take priority.
	// //src/parse:parse doesn't have 'go' but was explicitly requested so will be
	// added anyway.
	assert.Equal(t, state.ExpandOriginalLabels(), BuildLabels{
		{PackageName: "src/core", Name: "target1"},
		{PackageName: "src/parse", Name: "parse"},
	})
}

func TestExpandOriginalTestLabels(t *testing.T) {
	state := NewDefaultBuildState()
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core", Name: "all"}, true)
	state.NeedTests = true
	state.Include = []string{"go"}
	state.Exclude = []string{"py"}
	addTarget(state, "//src/core:target1", "go")
	addTarget(state, "//src/core:target2", "py")
	addTarget(state, "//src/core:target1_test", "go")
	addTarget(state, "//src/core:target2_test", "py")
	addTarget(state, "//src/core:target3_test")
	// Only the one target comes out here; it must be a test and otherwise follows
	// the same include / exclude logic as the previous test.
	assert.Equal(t, state.ExpandOriginalLabels(), BuildLabels{{PackageName: "src/core", Name: "target1_test"}})
}

func TestExpandVisibleOriginalTargets(t *testing.T) {
	state := NewDefaultBuildState()
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core", Name: "all"}, true)

	addTarget(state, "//src/core:target1", "py")
	addTarget(state, "//src/core:_target1#zip", "py")
	assert.Equal(t, state.ExpandVisibleOriginalTargets(), BuildLabels{{PackageName: "src/core", Name: "target1"}})
}

func TestExpandOriginalSubLabels(t *testing.T) {
	state := NewDefaultBuildState()
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core", Name: "..."}, true)
	state.Include = []string{"go"}
	state.Exclude = []string{"py"}
	addTarget(state, "//src/core:target1", "go")
	addTarget(state, "//src/core:target2", "py")
	addTarget(state, "//src/core/tests:target3", "go")
	// Only the one target comes out here; it must be a test and otherwise follows
	// the same include / exclude logic as the previous test.
	assert.Equal(t, state.ExpandOriginalLabels(), BuildLabels{
		{PackageName: "src/core", Name: "target1"},
		{PackageName: "src/core/tests", Name: "target3"},
	})
}

func TestExpandOriginalLabelsOrdering(t *testing.T) {
	state := NewDefaultBuildState()
	state.AddOriginalTarget(BuildLabel{PackageName: "src/parse", Name: "parse"}, true)
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core", Name: "..."}, true)
	state.AddOriginalTarget(BuildLabel{PackageName: "src/build", Name: "build"}, true)
	addTarget(state, "//src/core:target1", "go")
	addTarget(state, "//src/core:target2", "py")
	addTarget(state, "//src/core/tests:target3", "go")
	expected := BuildLabels{
		{PackageName: "src/parse", Name: "parse"},
		{PackageName: "src/core", Name: "target1"},
		{PackageName: "src/core", Name: "target2"},
		{PackageName: "src/core/tests", Name: "target3"},
		{PackageName: "src/build", Name: "build"},
	}
	assert.Equal(t, expected, state.ExpandOriginalLabels())
}

func TestAddDepsToTarget(t *testing.T) {
	state := NewDefaultBuildState()
	_, builds, _, _, _ := state.TaskQueues() //nolint:dogsled
	pkg := NewPackage("src/core")
	target1 := addTargetDeps(state, pkg, "//src/core:target1", "//src/core:target2")
	target2 := addTargetDeps(state, pkg, "//src/core:target2")
	state.Graph.AddPackage(pkg)
	state.QueueTarget(target1.Label, OriginalTarget, false, false)
	task := <-builds
	assert.Equal(t, target2.Label, task)
	// Now simulate target2 being built and adding a new dep to target1 in its post-build function.
	target3 := addTargetDeps(state, pkg, "//src/core:target3")
	target1.AddDependency(target3.Label)
	target2.FinishBuild()
	task = <-builds
	assert.Equal(t, target3.Label, task)
}

func addTarget(state *BuildState, name string, labels ...string) {
	target := NewBuildTarget(ParseBuildLabel(name, ""))
	target.Labels = labels
	target.IsTest = strings.HasSuffix(name, "_test")
	pkg := state.Graph.PackageByLabel(target.Label)
	if pkg == nil {
		pkg = NewPackage(target.Label.PackageName)
		state.Graph.AddPackage(pkg)
	}
	pkg.AddTarget(target)
	state.Graph.AddTarget(target)
}

func addTargetDeps(state *BuildState, pkg *Package, name string, deps ...string) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(name, ""))
	for _, d := range deps {
		target.AddDependency(ParseBuildLabel(d, ""))
	}
	pkg.AddTarget(target)
	state.Graph.AddTarget(target)
	return target
}
