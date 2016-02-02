// Tests on specific functions in build_target.go
package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanSee(t *testing.T) {
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

	// Deps should all be correct at this point
	assert.NoError(t, target1.CheckDependencyVisibility())
	assert.NoError(t, target2.CheckDependencyVisibility())
	assert.NoError(t, target3.CheckDependencyVisibility())
	assert.NoError(t, target4.CheckDependencyVisibility())
	assert.NoError(t, target5.CheckDependencyVisibility())
	assert.NoError(t, target6.CheckDependencyVisibility())
	assert.NoError(t, target7.CheckDependencyVisibility())

	// Now if we add a dep on this mock library, lib2 will fail because it's not a test.
	target2.Dependencies = append(target2.Dependencies, target5)
	assert.Error(t, target2.CheckDependencyVisibility())

	// Similarly to above test, if we add a dep on something that can't be seen, we should
	// get errors back from this function.
	target3.Dependencies = append(target3.Dependencies, target1)
	assert.Error(t, target3.CheckDependencyVisibility())
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
	target2 := makeTarget("//src/test:target2", "PUBLIC", target1)
	target2.AddOutput("//src/core:target1")
	target2.AddOutput("file3.go")
	target3 := makeTarget("//src/test:target3", "PUBLIC", target2)
	target3.AddOutput(":target2")
	target3.AddOutput("file4.go")

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

func TestCheckDuplicateOutputs(t *testing.T) {
	target1 := makeTarget("//src/core:target1", "PUBLIC")
	target3 := makeTarget("//src/core:target3", "PUBLIC")
	target2 := makeTarget("//src/core:target2", "PUBLIC", target1, target3)
	target1.AddOutput("thingy.txt")
	target3.AddOutput("thingy.txt")
	assert.NoError(t, target1.CheckDuplicateOutputs())
	target2.AddOutput(target1.Label.String())
	target2.AddOutput(target1.Label.String())
	// Not an error yet because AddOutput deduplicates trivially identical outputs.
	assert.NoError(t, target2.CheckDuplicateOutputs())
	// Will fail now we add the same output to another target.
	target2.AddOutput(target3.Label.String())
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
}

func makeTarget(label, visibility string, deps ...*BuildTarget) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(label, ""))
	if visibility == "PUBLIC" {
		target.Visibility = append(target.Visibility, BuildLabel{PackageName: "", Name: "..."})
	} else if visibility != "" {
		target.Visibility = append(target.Visibility, ParseBuildLabel(visibility, ""))
	}
	target.Dependencies = append(target.Dependencies, deps...)
	return target
}
