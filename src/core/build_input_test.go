package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNamedOutputLabel(t *testing.T) {
	pkg := NewPackage("")
	input := MustParseNamedOutputLabel("//src/core:target1|test", pkg)
	label, ok := input.(NamedOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test", label.Output)
}

func TestParseNamedOutputLabelRelative(t *testing.T) {
	pkg := NewPackage("src/core")
	input := MustParseNamedOutputLabel(":target1|test", pkg)
	label, ok := input.(NamedOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test", label.Output)
}

func TestParseNamedOutputLabelNoOutput(t *testing.T) {
	pkg := NewPackage("")
	input := MustParseNamedOutputLabel("//src/core:target1", pkg)
	_, ok := input.(NamedOutputLabel)
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
	_, ok := input.(NamedOutputLabel)
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
	_, ok := input.(NamedOutputLabel)
	assert.False(t, ok)
	label, ok := input.(BuildLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test_x86", label.Subrepo)
}
