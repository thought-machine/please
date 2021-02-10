package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNamedOutputLabel(t *testing.T) {
	pkg := NewPackage("")
	input := MustParseNamedOutputLabel("//src/core:target1|test", pkg)
	label, ok := input.(AnnotatedOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test", label.Annotation)
}

func TestParseNamedOutputLabelRelative(t *testing.T) {
	pkg := NewPackage("src/core")
	input := MustParseNamedOutputLabel(":target1|test", pkg)
	label, ok := input.(AnnotatedOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test", label.Annotation)
}

func TestParseNamedOutputLabelNoOutput(t *testing.T) {
	pkg := NewPackage("")
	input := MustParseNamedOutputLabel("//src/core:target1", pkg)
	_, ok := input.(AnnotatedOutputLabel)
	assert.False(t, ok)
	label, ok := input.(BuildLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
}

func TestParseNamedOutputLabelEmptyOutput(t *testing.T) {
	pkg := NewPackage("")
	assert.Panics(t, func() {
		MustParseNamedOutputLabel("//src/core:target1|", pkg)
	})
}

func TestParseNamedOutputLabelSubrepo(t *testing.T) {
	pkg := NewPackage("")
	input := MustParseNamedOutputLabel("@test_x86//src/core:target1", pkg)
	_, ok := input.(AnnotatedOutputLabel)
	assert.False(t, ok)
	label, ok := input.(BuildLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test_x86", label.Subrepo)
}

func TestParseNamedOutputLabelRelativeSubrepo(t *testing.T) {
	pkg := NewPackage("src/core")
	input := MustParseNamedOutputLabel("@test_x86:target1", pkg)
	_, ok := input.(AnnotatedOutputLabel)
	assert.False(t, ok)
	label, ok := input.(BuildLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test_x86", label.Subrepo)
}

func TestStringifyAnnotatedOutputLabel(t *testing.T) {
	l := AnnotatedOutputLabel{BuildLabel: BuildLabel{PackageName: "src/core", Name: "build_input_test"}}
	assert.Equal(t, "//src/core:build_input_test", l.String())
	l.Annotation = "thing"
	assert.Equal(t, "//src/core:build_input_test|thing", l.String())
}
