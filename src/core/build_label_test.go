package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncludes(t *testing.T) {
	label1 := BuildLabel{PackageName: "src/core", Name: "..."}
	label2 := BuildLabel{PackageName: "src/parse", Name: "parse"}
	assert.False(t, label1.includes(label2))
	label2 = BuildLabel{PackageName: "src/core", Name: "core_test"}
	assert.True(t, label1.includes(label2))
}

func TestIncludesSubstring(t *testing.T) {
	label1 := BuildLabel{PackageName: "third_party/python", Name: "..."}
	label2 := BuildLabel{PackageName: "third_party/python3", Name: "six"}
	assert.False(t, label1.includes(label2))
}

func TestIncludesSubpackages(t *testing.T) {
	label1 := BuildLabel{PackageName: "", Name: "..."}
	label2 := BuildLabel{PackageName: "third_party/python3", Name: "six"}
	assert.True(t, label1.includes(label2))
}

func TestParent(t *testing.T) {
	label := BuildLabel{PackageName: "src/core", Name: "core"}
	assert.Equal(t, label, label.Parent())
	label2 := BuildLabel{PackageName: "src/core", Name: "_core#src"}
	assert.Equal(t, label, label2.Parent())
	label3 := BuildLabel{PackageName: "src/core", Name: "_core_something"}
	assert.Equal(t, label3, label3.Parent())
}
