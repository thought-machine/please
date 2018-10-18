package intellij

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)


func TestPathFromModuleFileToInputsSimple(t *testing.T) {

	inputs := []core.BuildInput{
		core.FileLabel{
			File:    "File1.java",
			Package: "some/simple/path",
		},
		core.FileLabel{
			File:    "File2.java",
			Package: "some/simple/path",
		},
	}

	expected := "some/simple/path"
	assert.Equal(t, expected, *commonDirectoryFromInputs(nil, inputs))
}

func TestPathFromModuleFileToInputsComplex(t *testing.T) {

	inputs := []core.BuildInput{
		core.FileLabel{
			File:    "File1.java",
			Package: "some/not_so_simple/path",
		},
		core.FileLabel{
			File:    "File2.java",
			Package: "some/simple/path",
		},
	}

	expected := "some"
	assert.Equal(t, expected, *commonDirectoryFromInputs(nil, inputs))
}

func TestPathFromModuleFileToInputsNoMatch(t *testing.T) {

	inputs := []core.BuildInput{
		core.FileLabel{
			File:    "File1.java",
			Package: "some/simple/path",
		},
		core.FileLabel{
			File:    "File2.java",
			Package: "another/simple/path",
		},
	}

	expected := "."
	assert.Equal(t, expected, *commonDirectoryFromInputs(nil, inputs))
}

func TestProjectLocation(t *testing.T) {
	assert.True(t, strings.HasSuffix(projectLocation(), "plz-out/intellij/.idea"))
}

func TestModuleFileLocation(t *testing.T) {
	target := &core.BuildTarget{
		Label: core.BuildLabel{
			PackageName: "some/package", Name: "target", Subrepo: "",
		},
	}

	assert.True(t, strings.HasSuffix(moduleFileLocation(target), "plz-out/intellij/some/package/target.iml"))
}