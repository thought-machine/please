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
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core", Name: "all"}, true)
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core/tests", Name: "all"}, true)
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
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core", Name: "all"}, true)
	state.AddOriginalTarget(BuildLabel{PackageName: "src/core/tests", Name: "all"}, true)
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

func TestAddTargetFilegroupPackageOutputs(t *testing.T) {
	state := NewDefaultBuildState()

	pkg := NewPackage("src/core")
	target := NewBuildTarget(ParseBuildLabel("//src/core:test", ""))
	target.IsFilegroup = true
	target.AddSource(NewFileLabel("file.txt", pkg))
	pkg.AddTarget(target)

	state.AddTarget(pkg, target)
	assert.Len(t, pkg.Outputs, 1)
	// This tests that the output shouldn't include the package name
	_, exists := pkg.Outputs["file.txt"]
	assert.True(t, exists)
}

func TestAddDepsToTarget(t *testing.T) {
	state := NewDefaultBuildState()
	_, builds := state.TaskQueues()
	pkg := NewPackage("src/core")
	target1 := addTargetDeps(state, pkg, "//src/core:target1", "//src/core:target2")
	target2 := addTargetDeps(state, pkg, "//src/core:target2")
	state.Graph.AddPackage(pkg)
	state.QueueTarget(target1.Label, OriginalTarget, false, ParseModeNormal)
	task := <-builds
	assert.Equal(t, Task{Target: target2}, task)
	// Now simulate target2 being built and adding a new dep to target1 in its post-build function.
	target3 := addTargetDeps(state, pkg, "//src/core:target3")
	target1.AddDependency(target3.Label)
	target2.FinishBuild()
	task = <-builds
	assert.Equal(t, Task{Target: target3}, task)
}

func addTarget(state *BuildState, name string, labels ...string) {
	target := NewBuildTarget(ParseBuildLabel(name, ""))
	target.Labels = labels
	if strings.HasSuffix(name, "_test") {
		target.Test = new(TestFields)
	}
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

func TestCopyPlugin(t *testing.T) {
	plugin := &Plugin{
		ExtraValues: map[string][]string{
			"foo": {"foo"},
		},
	}

	newPlugin := plugin.copyPlugin()

	assert.False(t, plugin == newPlugin)

	newPlugin.ExtraValues["foo"] = []string{"bar"}

	assert.NotEqual(t, plugin.ExtraValues["foo"], newPlugin.ExtraValues["foo"])
}
