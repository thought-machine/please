package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterSubinclude(t *testing.T) {
	pkg := NewPackage("src/core")
	label1 := ParseBuildLabel("//build_defs:js", "")
	label2 := ParseBuildLabel("//build_defs:go", "")
	pkg.RegisterSubinclude(label1)
	pkg.RegisterSubinclude(label2)
	pkg.RegisterSubinclude(label1)
	assert.Equal(t, []BuildLabel{label1, label2}, pkg.Subincludes)
}

func TestRegisterOutput(t *testing.T) {
	state := NewDefaultBuildState()
	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	pkg := NewPackage("src/core")
	target1.Sources = append(target1.Sources, FileLabel{File: "file1.go"})
	target2.Sources = append(target2.Sources, FileLabel{File: "file2.go"})
	target2.AddNamedSource("go", FileLabel{File: "file1.go"})

	assert.NoError(t, pkg.RegisterOutput(state, "file1.go", target1))
	assert.NoError(t, pkg.RegisterOutput(state, "file2.go", target2))
	assert.Error(t, pkg.RegisterOutput(state, "file1.go", target2))

	assert.NoError(t, pkg.RegisterOutput(state, "file3.go", target1))
	assert.Error(t, pkg.RegisterOutput(state, "file3.go", target2))
}

func TestRegisterOutputNonFilegroupTargets(t *testing.T) {
	state := NewDefaultBuildState()
	state.Config.FeatureFlags.PackageOutputsStrictness = true

	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	pkg := NewPackage("src/core")

	assert.NoError(t, pkg.RegisterOutput(state, "file.go", target1))
	assert.Error(t, pkg.RegisterOutput(state, "file.go", target2))
}

func TestRegisterOutputFilegroupAndNonFilegroupTargets(t *testing.T) {
	state := NewDefaultBuildState()
	state.Config.FeatureFlags.PackageOutputsStrictness = true

	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	target2.IsFilegroup = true
	pkg := NewPackage("src/core")

	assert.NoError(t, pkg.RegisterOutput(state, "file1.go", target1))
	assert.Error(t, pkg.RegisterOutput(state, "file1.go", target2))

	assert.NoError(t, pkg.RegisterOutput(state, "file2.go", target2))
	assert.Error(t, pkg.RegisterOutput(state, "file2.go", target1))
}

func TestRegisterOutputFilegroupTargets(t *testing.T) {
	state := NewDefaultBuildState()
	state.Config.FeatureFlags.PackageOutputsStrictness = true

	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target1.IsFilegroup = true
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	target2.IsFilegroup = true
	pkg := NewPackage("src/core")

	// The same local file can be registered if coming from filegroups
	assert.NoError(t, pkg.RegisterOutput(state, "file.go", target1))
	assert.NoError(t, pkg.RegisterOutput(state, "file.go", target2))
}

func TestAllChildren(t *testing.T) {
	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	target2a := NewBuildTarget(ParseBuildLabel("//src/core:_target2#a", ""))
	pkg := NewPackage("src/core")
	pkg.AddTarget(target1)
	pkg.AddTarget(target2)
	pkg.AddTarget(target2a)
	children := pkg.AllChildren(target2)
	expected := []*BuildTarget{target2a, target2}
	assert.Equal(t, expected, children)
	children = pkg.AllChildren(target2a)
	assert.Equal(t, expected, children)
}

func TestFindOwningPackages(t *testing.T) {
	state := NewDefaultBuildState()
	state.Config.Parse.BuildFileName = []string{"BUILD_FILE"}
	pkgs := FindOwningPackages(state, []string{"src/core/test_data/test_subfolder1/whatever.txt"})
	assert.Equal(t, []BuildLabel{ParseBuildLabel("//src/core/test_data:all", "")}, pkgs)
}

func TestIsIncludedIn(t *testing.T) {
	label := BuildLabel{PackageName: "src", Name: "..."}
	assert.True(t, NewPackage("src").IsIncludedIn(label))
	assert.True(t, NewPackage("src/core").IsIncludedIn(label))
	assert.False(t, NewPackage("src2").IsIncludedIn(label))
}

func TestVerifyOutputs(t *testing.T) {
	state := NewDefaultBuildState()
	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	pkg := NewPackage("src/core")
	pkg.AddTarget(target1)
	pkg.AddTarget(target2)
	pkg.MustRegisterOutput(state, "dir/file1.go", target1)
	pkg.MustRegisterOutput(state, "dir", target2)
	assert.Equal(t, 1, len(pkg.verifyOutputs()))
	target1.AddDependency(target2.Label)
	assert.Equal(t, 0, len(pkg.verifyOutputs()))
}

func TestSuggestNoTargetFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "wobble", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target2"), bl("//src/core:wibble"))
	assert.Equal(t, s, "", "No suggestion because they're not similar at all.")
}

func TestSuggestSingleTargetFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target2"), bl("//src/core:wibble"))
	assert.Equal(t, s, "\nMaybe you meant :target1 ?")
}

func TestSuggestTwoTargetsFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/core:blibble"))
	assert.Equal(t, s, "\nMaybe you meant :target1 or :target21 ?")
}

func TestSuggestSeveralTargetsFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "target_21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/core:blibble"))
	assert.Equal(t, s, "\nMaybe you meant :target1 , :target21 or :target_21 ?")
}

func TestSuggestSingleTargetFromAnotherPackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target2"), bl("//src/parse:wibble"))
	assert.Equal(t, s, "\nMaybe you meant //src/core:target1 ?")
}

func TestSuggestTwoTargetsFromAnotherPackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/parse:blibble"))
	assert.Equal(t, s, "\nMaybe you meant //src/core:target1 or //src/core:target21 ?")
}

func TestSuggestSeveralTargetsFromAnotherPackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "target_21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/parse:blibble"))
	assert.Equal(t, s, "\nMaybe you meant //src/core:target1 , //src/core:target21 or //src/core:target_21 ?")
}

func makePackage(name string, targets ...string) *Package {
	pkg := NewPackage(name)
	for _, target := range targets {
		pkg.AddTarget(NewBuildTarget(bl("//" + name + ":" + target)))
	}
	return pkg
}

func bl(label string) BuildLabel {
	return ParseBuildLabel(label, "")
}
