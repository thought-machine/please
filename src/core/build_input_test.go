package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBuildOutputLabel(t *testing.T) {
	input, err := TryParseBuildOutputLabel("//src/core:target1|test.go", "")
	assert.NoError(t, err)
	label, ok := input.(BuildOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test.go", label.Output)
}

func TestParseBuildOutputLabelRelative(t *testing.T) {
	input, err := TryParseBuildOutputLabel(":target1|test.go", "src/core")
	assert.NoError(t, err)
	label, ok := input.(BuildOutputLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
	assert.Equal(t, "test.go", label.Output)
}

func TestParseBuildOutputLabelNoOutput(t *testing.T) {
	input, err := TryParseBuildOutputLabel("//src/core:target1", "")
	assert.NoError(t, err)
	_, ok := input.(BuildOutputLabel)
	assert.False(t, ok)
	label, ok := input.(BuildLabel)
	assert.True(t, ok)
	assert.Equal(t, "src/core", label.PackageName)
	assert.Equal(t, "target1", label.Name)
}

func TestParseBuildOutputLabelEmptyOutput(t *testing.T) {
	_, err := TryParseBuildOutputLabel("//src/core:target1|", "")
	assert.Error(t, err)
}
