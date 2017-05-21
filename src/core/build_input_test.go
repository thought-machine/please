package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNamedOutputLabel(t *testing.T) {
	input, err := TryParseNamedOutputLabel("//src/core:target1|test", "")
	assert.NoError(t, err)
	label, ok := input.(NamedOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test", label.Output)
}

func TestParseNamedOutputLabelRelative(t *testing.T) {
	input, err := TryParseNamedOutputLabel(":target1|test", "src/core")
	assert.NoError(t, err)
	label, ok := input.(NamedOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test", label.Output)
}

func TestParseNamedOutputLabelNoOutput(t *testing.T) {
	input, err := TryParseNamedOutputLabel("//src/core:target1", "")
	assert.NoError(t, err)
	_, ok := input.(NamedOutputLabel)
	assert.False(t, ok)
	label, ok := input.(BuildLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
}

func TestParseNamedOutputLabelEmptyOutput(t *testing.T) {
	_, err := TryParseNamedOutputLabel("//src/core:target1|", "")
	assert.Error(t, err)
}
