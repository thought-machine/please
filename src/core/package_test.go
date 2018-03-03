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
	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	pkg := NewPackage("src/core")
	target1.Sources = append(target1.Sources, FileLabel{File: "file1.go"})
	target2.Sources = append(target2.Sources, FileLabel{File: "file2.go"})
	target2.AddNamedSource("go", FileLabel{File: "file1.go"})
	pkg.RegisterOutput("file1.go", target1)
	pkg.RegisterOutput("file2.go", target2)
	// Doesn't panic because it's a source of both rules, so we assume it's a filegroup.
	pkg.RegisterOutput("file1.go", target2)

	pkg.RegisterOutput("file3.go", target1)
	assert.Error(t, pkg.RegisterOutput("file3.go", target2))
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
	state.Config.Parse.BuildFileName = []string{"TEST_BUILD"}
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
	target1 := NewBuildTarget(ParseBuildLabel("//src/core:target1", ""))
	target2 := NewBuildTarget(ParseBuildLabel("//src/core:target2", ""))
	pkg := NewPackage("src/core")
	pkg.AddTarget(target1)
	pkg.AddTarget(target2)
	pkg.MustRegisterOutput("dir/file1.go", target1)
	pkg.MustRegisterOutput("dir", target2)
	assert.Equal(t, 1, len(pkg.verifyOutputs()))
	target1.AddDependency(target2.Label)
	assert.Equal(t, 0, len(pkg.verifyOutputs()))
}
