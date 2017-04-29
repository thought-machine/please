// Tests on specific functions in build_target.go
package core

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanSee(t *testing.T) {
	NewBuildState(1, nil, 1, DefaultConfiguration())
	target1 := makeTarget("//src/build/python:lib1", "")
	target2 := makeTarget("//src/build/python:lib2", "PUBLIC")
	target3 := makeTarget("//src/test/python:lib3", "//src/test/...")
	target4 := makeTarget("//src/test/python/moar:lib4", "")

	// target2 is public so anything can import it
	assert.True(t, target3.CanSee(target2), "couldn't see public target")
	assert.True(t, target4.CanSee(target2), "couldn't see public target")

	// target1 is not public, other things can't import it
	assert.False(t, target3.CanSee(target1), "could see private target")
	assert.False(t, target4.CanSee(target1), "could see private target")

	// target2 is in the same package as target1 so can implicitly see it
	assert.True(t, target2.CanSee(target1), "couldn't see target in the same package")

	// target3 is only visible to a subtree, things outside it can't see it
	assert.False(t, target1.CanSee(target3), "could see target with restricted visibility")
	assert.False(t, target2.CanSee(target3), "could see target with restricted visibility")

	// Targets in that subtree can though.
	assert.True(t, target4.CanSee(target3), "couldn't see target within its visibility spec")

	// Sub-targets can see things visible to their parents.
	target5 := makeTarget("//src/build/python:lib5", "//src/test/python:test5")
	target6 := makeTarget("//src/test/python:_test5#pex", "")
	assert.True(t, target6.CanSee(target5))
	assert.False(t, target5.CanSee(target6))
}

func TestCanSeeExperimental(t *testing.T) {
	config := DefaultConfiguration()
	config.Please.ExperimentalDir = "experimental"
	NewBuildState(1, nil, 1, config)

	target1 := makeTarget("//src/core:target1", "")
	target2 := makeTarget("//experimental/user:target2", "PUBLIC")

	// target2 can see target1 since it's in experimental, which suppress normal visibility constraints.
	assert.True(t, target2.CanSee(target1))
	// target1 can't see target2 because it's in experimental, even though it's public.
	assert.False(t, target1.CanSee(target2))
}

func TestCheckDependencyVisibility(t *testing.T) {
	target1 := makeTarget("//src/build/python:lib1", "")
	target2 := makeTarget("//src/build/python:lib2", "PUBLIC", target1)
	target3 := makeTarget("//src/test/python:lib3", "//src/test/...", target2)
	target4 := makeTarget("//src/test/python/moar:lib4", "//src/test/...", target3)
	target5 := makeTarget("//third_party/python:mock", "PUBLIC")
	target5.TestOnly = true
	target6 := makeTarget("//src/test/python:test_lib", "", target5)
	target6.TestOnly = true
	target7 := makeTarget("//src/test/python:test1", "", target5, target4)
	target7.IsTest = true

	graph := NewGraph()
	graph.AddTarget(target1)
	graph.AddTarget(target2)
	graph.AddTarget(target3)
	graph.AddTarget(target4)
	graph.AddTarget(target5)
	graph.AddTarget(target6)
	graph.AddTarget(target7)

	// Deps should all be correct at this point
	assert.NoError(t, target1.CheckDependencyVisibility(graph))
	assert.NoError(t, target2.CheckDependencyVisibility(graph))
	assert.NoError(t, target3.CheckDependencyVisibility(graph))
	assert.NoError(t, target4.CheckDependencyVisibility(graph))
	assert.NoError(t, target5.CheckDependencyVisibility(graph))
	assert.NoError(t, target6.CheckDependencyVisibility(graph))
	assert.NoError(t, target7.CheckDependencyVisibility(graph))

	// Now if we add a dep on this mock library, lib2 will fail because it's not a test.
	target2.resolveDependency(target5.Label, target5)
	assert.Error(t, target2.CheckDependencyVisibility(graph))

	// Similarly to above test, if we add a dep on something that can't be seen, we should
	// get errors back from this function.
	target3.resolveDependency(target1.Label, target1)
	assert.Error(t, target3.CheckDependencyVisibility(graph))
}

func TestAddOutput(t *testing.T) {
	target := makeTarget("//src/test/python:lib1", "")
	target.AddOutput("thingy.py")
	target.AddOutput("thingy2.py")
	target.AddOutput("thingy.py")
	if len(target.Outputs()) != 2 {
		t.Errorf("Incorrect output length; should be 2, was %d", len(target.Outputs()))
	}
}

func TestAddOutputSorting(t *testing.T) {
	target := makeTarget("//src/test/python:lib1", "")
	target.AddOutput("1.py")
	target.AddOutput("2.py")
	target.AddOutput("1.py")
	target.AddOutput("3.py")
	target.AddOutput("1.pyx")
	target.AddOutput("x.pyx")
	expected := []string{
		"1.py",
		"1.pyx",
		"2.py",
		"3.py",
		"x.pyx",
	}
	assert.Equal(t, expected, target.Outputs())
}

func TestAddSource(t *testing.T) {
	target := makeTarget("//src/test/python:lib1", "")
	target.AddSource(ParseBuildLabel("//src/test/python:lib2", ""))
	target.AddSource(ParseBuildLabel("//src/test/python:lib3", ""))
	target.AddSource(ParseBuildLabel("//src/test/python:lib2", ""))
	assert.Equal(t, 2, len(target.Sources))
	assert.Equal(t, 2, len(target.DeclaredDependencies()))
}

func TestSetContainerSettings(t *testing.T) {
	target := makeTarget("//src/test/python:lib1", "")

	target.SetContainerSetting("dockerimage", "tm/special_image:v2")
	assert.Equal(t, "tm/special_image:v2", target.ContainerSettings.DockerImage)

	target.SetContainerSetting("dockeruser", "")
	assert.Equal(t, "", target.ContainerSettings.DockerUser)

	target.SetContainerSetting("dockerrunargs", "-it")
	assert.Equal(t, "-it", target.ContainerSettings.DockerRunArgs)
}

func TestOutputs(t *testing.T) {
	target1 := makeTarget("//src/core:target1", "PUBLIC")
	target1.AddOutput("file1.go")
	target1.AddOutput("file2.go")
	target2 := makeFilegroup("//src/test:target2", "PUBLIC", target1)
	target2.AddSource(target1.Label)
	addFilegroupSource(target2, "file3.go")
	target3 := makeFilegroup("//src/test:target3", "PUBLIC", target2)
	target3.AddSource(target2.Label)
	addFilegroupSource(target3, "file4.go")

	assert.Equal(t, []string{"file1.go", "file2.go"}, target1.Outputs())
	assert.Equal(t, []string{"file1.go", "file2.go", "file3.go"}, target2.Outputs())
	assert.Equal(t, []string{"file1.go", "file2.go", "file3.go", "file4.go"}, target3.Outputs())
}

func TestProvideFor(t *testing.T) {
	// target1 is provided directly since they have a simple dependency
	target1 := makeTarget("//src/core:target1", "PUBLIC")
	target2 := makeTarget("//src/core:target2", "PUBLIC", target1)
	assert.Equal(t, []BuildLabel{target1.Label}, target1.ProvideFor(target2))
	// Now have target2 provide target1. target3 will get target1 instead.
	target2.Provides = map[string]BuildLabel{"whatevs": target1.Label}
	target3 := makeTarget("//src/core:target3", "PUBLIC", target2)
	target3.Requires = append(target3.Requires, "whatevs")
	assert.Equal(t, []BuildLabel{target1.Label}, target2.ProvideFor(target3))
	// Now target4 has a data dependency on target2. It has the same requirement as target3 but
	// it gets target2 instead of target1, because that's just how data deps work.
	target4 := makeTarget("//src/core:target4", "PUBLIC", target2)
	target4.Data = append(target4.Data, target2.Label)
	target4.Requires = append(target4.Requires, "whatevs")
	assert.Equal(t, []BuildLabel{target2.Label}, target2.ProvideFor(target4))
}

func TestAddProvide(t *testing.T) {
	target1 := makeTarget("//src/core:target1", "PUBLIC")
	target2 := makeTarget("//src/core:target2", "PUBLIC", target1)
	target3 := makeTarget("//src/core:target3", "PUBLIC", target2)
	target2.AddDependency(target1.Label)
	target2.AddProvide("go", ParseBuildLabel(":target1", "src/core"))
	target3.Requires = append(target3.Requires, "go")
	assert.Equal(t, []BuildLabel{target1.Label}, target2.ProvideFor(target3))
}

func TestCheckDuplicateOutputs(t *testing.T) {
	target1 := makeFilegroup("//src/core:target1", "PUBLIC")
	target3 := makeFilegroup("//src/core:target3", "PUBLIC")
	target2 := makeFilegroup("//src/core:target2", "PUBLIC", target1, target3)
	addFilegroupSource(target1, "thingy.txt")
	addFilegroupSource(target3, "thingy.txt")
	assert.NoError(t, target1.CheckDuplicateOutputs())
	target2.AddSource(target1.Label)
	target2.AddSource(target1.Label)
	// Not an error yet because AddOutput deduplicates trivially identical outputs.
	assert.NoError(t, target2.CheckDuplicateOutputs())
	// Will fail now we add the same output to another target.
	target2.AddSource(target3.Label)
	assert.Error(t, target2.CheckDuplicateOutputs())
}

func TestLabels(t *testing.T) {
	target := makeTarget("//src/core:target1", "PUBLIC")
	assert.False(t, target.HasLabel("py"))
	assert.False(t, target.HasAnyLabel([]string{"py", "go"}))
	target.AddLabel("py")
	assert.True(t, target.HasLabel("py"))
	assert.True(t, target.HasAnyLabel([]string{"py", "go"}))
	target.AddLabel("py")
	target.AddLabel("go")
	assert.Equal(t, 2, len(target.Labels))
	// "test" label is implicit on tests.
	assert.False(t, target.HasLabel("test"))
	target.IsTest = true
	assert.True(t, target.HasLabel("test"))
}

func TestGetCommandConfig(t *testing.T) {
	target := makeTarget("//src/core:target1", "PUBLIC")
	target.Command = "test1"
	assert.Equal(t, "test1", target.GetCommandConfig(""))
	target.Command = ""
	target.AddCommand("opt", "test3")
	target.AddCommand("dbg", "test4")
	assert.Equal(t, "test3", target.GetCommandConfig("opt"))
	assert.Equal(t, "test4", target.GetCommandConfig("dbg"))
}

func TestGetCommand(t *testing.T) {
	state := NewBuildState(10, nil, 2, DefaultConfiguration())
	state.Config.Build.Config = "dbg"
	state.Config.Build.FallbackConfig = "opt"
	target := makeTarget("//src/core:target1", "PUBLIC")
	target.Command = "test1"
	assert.Equal(t, "test1", target.GetCommand())
	assert.Panics(t, func() { target.AddCommand("opt", "test2") },
		"Should panic when adding a config command to a target with a command already")
	target.Command = ""
	target.AddCommand("opt", "test3")
	target.AddCommand("dbg", "test4")
	assert.Equal(t, "test4", target.GetCommand(), "Current config is dbg")
	state.Config.Build.Config = "opt"
	assert.Equal(t, "test3", target.GetCommand(), "Current config is opt")
	state.Config.Build.Config = "fast"
	assert.Equal(t, "test3", target.GetCommand(), "Default config is opt, should fall back to that")
}

func TestGetTestCommand(t *testing.T) {
	state := NewBuildState(10, nil, 2, DefaultConfiguration())
	state.Config.Build.Config = "dbg"
	state.Config.Build.FallbackConfig = "opt"
	target := makeTarget("//src/core:target1", "PUBLIC")
	target.TestCommand = "test1"
	assert.Equal(t, "test1", target.GetTestCommand())
	assert.Panics(t, func() { target.AddTestCommand("opt", "test2") },
		"Should panic when adding a config command to a target with a command already")
	target.TestCommand = ""
	target.AddTestCommand("opt", "test3")
	target.AddTestCommand("dbg", "test4")
	assert.Equal(t, "test4", target.GetTestCommand(), "Current config is dbg")
	state.Config.Build.Config = "opt"
	assert.Equal(t, "test3", target.GetTestCommand(), "Current config is opt")
	state.Config.Build.Config = "fast"
	assert.Equal(t, "test3", target.GetTestCommand(), "Default config is opt, should fall back to that")
}

func TestHasSource(t *testing.T) {
	target := makeTarget("//src/core:target1", "")
	target.Sources = append(target.Sources, FileLabel{File: "file1.go"})
	target.AddNamedSource("wevs", FileLabel{File: "file2.go"})
	assert.True(t, target.HasSource("file1.go"))
	assert.True(t, target.HasSource("file2.go"))
	assert.False(t, target.HasSource("file3.go"))
}

func TestToolPath(t *testing.T) {
	target := makeTarget("//src/core:target1", "")
	target.AddOutput("file1.go")
	target.AddOutput("file2.go")
	wd, _ := os.Getwd()
	root := wd + "/plz-out/gen/src/core"
	assert.Equal(t, fmt.Sprintf("%s/file1.go %s/file2.go", root, root), target.toolPath())
}

func TestDependencies(t *testing.T) {
	target1 := makeTarget("//src/core:target1", "")
	target2 := makeTarget("//src/core:target2", "", target1)
	target3 := makeTarget("//src/core:target3", "", target1, target2)
	assert.Equal(t, []BuildLabel{}, target1.DeclaredDependencies())
	assert.Equal(t, []*BuildTarget{}, target1.Dependencies())
	assert.Equal(t, []BuildLabel{target1.Label}, target2.DeclaredDependencies())
	assert.Equal(t, []*BuildTarget{target1}, target2.Dependencies())
	assert.Equal(t, []BuildLabel{target1.Label, target2.Label}, target3.DeclaredDependencies())
	assert.Equal(t, []*BuildTarget{target1, target2}, target3.Dependencies())
}

func TestDependenciesExported(t *testing.T) {
	target1 := makeTarget("//src/core:target1", "")
	target2 := makeTarget("//src/core:target2", "")
	target2.AddMaybeExportedDependency(target1.Label, true)
	target2.resolveDependency(target1.Label, target1)
	target3 := makeTarget("//src/core:target3", "", target2)
	assert.Equal(t, []BuildLabel{}, target1.DeclaredDependencies())
	assert.Equal(t, []*BuildTarget{}, target1.Dependencies())
	assert.Equal(t, []BuildLabel{target1.Label}, target2.DeclaredDependencies())
	assert.Equal(t, []*BuildTarget{target1}, target2.Dependencies())
	assert.Equal(t, []BuildLabel{target2.Label}, target3.DeclaredDependencies())
	assert.Equal(t, []*BuildTarget{target1, target2}, target3.Dependencies())
}

func TestAddDependency(t *testing.T) {
	target1 := makeTarget("//src/core:target1", "")
	target2 := makeTarget("//src/core:target2", "")
	assert.Equal(t, []BuildLabel{}, target2.DeclaredDependencies())
	assert.Equal(t, []BuildLabel{}, target2.ExportedDependencies())
	target2.AddDependency(target1.Label)
	assert.Equal(t, []BuildLabel{target1.Label}, target2.DeclaredDependencies())
	assert.Equal(t, []BuildLabel{}, target2.ExportedDependencies())
	target2.AddMaybeExportedDependency(target1.Label, true)
	assert.Equal(t, []BuildLabel{target1.Label}, target2.DeclaredDependencies())
	assert.Equal(t, []BuildLabel{target1.Label}, target2.ExportedDependencies())
	assert.Equal(t, []*BuildTarget{}, target2.Dependencies())
	target2.resolveDependency(target1.Label, target1)
	assert.Equal(t, []*BuildTarget{target1}, target2.Dependencies())
}

func TestDependencyFor(t *testing.T) {
	target1 := makeTarget("//src/core:target1", "")
	target2 := makeTarget("//src/core:target2", "", target1)
	assert.Equal(t, []*BuildTarget{target1}, target2.DependenciesFor(target1.Label))
	assert.Equal(t, []*BuildTarget(nil), target2.DependenciesFor(target2.Label))
	assert.Equal(t, 1, len(target2.dependencies))
}

func TestParent(t *testing.T) {
	// "grandchild" is of course a misnomer since we only really have a concept of
	// one level of parent-child relationship.
	grandchild := makeTarget("//src/core:__target1#child#grandchild", "")
	child := makeTarget("//src/core:_target1#child", "", grandchild)
	parent := makeTarget("//src/core:target1", "", child)
	graph := NewGraph()
	graph.AddTarget(grandchild)
	graph.AddTarget(child)
	graph.AddTarget(parent)

	assert.Equal(t, parent.Label, grandchild.Label.Parent())
	assert.Equal(t, parent.Label, child.Label.Parent())
	assert.Equal(t, parent.Label, parent.Label.Parent())
	assert.Equal(t, parent, grandchild.Parent(graph))
	assert.Equal(t, parent, child.Parent(graph))
	assert.Equal(t, (*BuildTarget)(nil), parent.Parent(graph))
}

func TestHasParent(t *testing.T) {
	grandchild := makeTarget("//src/core:__target1#child#grandchild", "")
	child := makeTarget("//src/core:_target1#child", "", grandchild)
	parent := makeTarget("//src/core:target1", "", child)
	assert.True(t, grandchild.HasParent())
	assert.True(t, child.HasParent())
	assert.False(t, parent.HasParent())
}

func TestOutMode(t *testing.T) {
	// Check that output modes match the binary flag correctly.
	// This feels a little fatuous but it's hard to have any less specific assertions on it.
	target := makeTarget("//src/core:target1", "")
	assert.Equal(t, os.FileMode(0444), target.OutMode())
	target.IsBinary = true
	assert.Equal(t, os.FileMode(0555), target.OutMode())
}

func TestOutputOrdering(t *testing.T) {
	// Check that outputs come out ordered, this is important for hash stability; previously
	// we preseved the original order, but tools like buildifier may reorder them assuming
	// that the order of arguments is not important.
	target1 := makeTarget("//src/core:target1", "")
	target1.AddOutput("file1.txt")
	target1.AddOutput("file2.txt")
	target2 := makeTarget("//src/core:target2", "")
	target2.AddOutput("file2.txt")
	target2.AddOutput("file1.txt")
	assert.Equal(t, target1.DeclaredOutputs(), target2.DeclaredOutputs())
	assert.Equal(t, target1.Outputs(), target2.Outputs())
}

func TestAllLocalSources(t *testing.T) {
	target := makeTarget("//src/core:target1", "")
	target.AddSource(FileLabel{File: "target1.go", Package: "src/core"})
	target.AddSource(BuildLabel{Name: "target2", PackageName: "src/core"})
	target.AddSource(SystemFileLabel{Path: "/usr/bin/bash"})
	assert.Equal(t, []string{"src/core/target1.go"}, target.AllLocalSources())
}

func makeTarget(label, visibility string, deps ...*BuildTarget) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(label, ""))
	if visibility == "PUBLIC" {
		target.Visibility = append(target.Visibility, BuildLabel{PackageName: "", Name: "..."})
	} else if visibility != "" {
		target.Visibility = append(target.Visibility, ParseBuildLabel(visibility, ""))
	}
	for _, dep := range deps {
		target.AddDependency(dep.Label)
		target.resolveDependency(dep.Label, dep)
	}
	return target
}

func makeFilegroup(label, visibility string, deps ...*BuildTarget) *BuildTarget {
	target := makeTarget(label, visibility, deps...)
	target.IsFilegroup = true
	return target
}

func addFilegroupSource(target *BuildTarget, source string) {
	target.AddSource(FileLabel{Package: target.Label.PackageName, File: source})
}
