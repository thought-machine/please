package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncludes(t *testing.T) {
	label1 := BuildLabel{PackageName: "src/core", Name: "..."}
	label2 := BuildLabel{PackageName: "src/parse", Name: "parse"}
	assert.False(t, label1.Includes(label2))
	label2 = BuildLabel{PackageName: "src/core", Name: "core_test"}
	assert.True(t, label1.Includes(label2))
}

func TestIncludesSubstring(t *testing.T) {
	label1 := BuildLabel{PackageName: "third_party/python", Name: "..."}
	label2 := BuildLabel{PackageName: "third_party/python3", Name: "six"}
	assert.False(t, label1.Includes(label2))
}

func TestIncludesSubpackages(t *testing.T) {
	label1 := BuildLabel{PackageName: "", Name: "..."}
	label2 := BuildLabel{PackageName: "third_party/python3", Name: "six"}
	assert.True(t, label1.Includes(label2))
}

func TestParent(t *testing.T) {
	label := BuildLabel{PackageName: "src/core", Name: "core"}
	assert.Equal(t, label, label.Parent())
	label2 := BuildLabel{PackageName: "src/core", Name: "_core#src"}
	assert.Equal(t, label, label2.Parent())
	label3 := BuildLabel{PackageName: "src/core", Name: "_core_something"}
	assert.Equal(t, label3, label3.Parent())
}

func TestUnmarshalFlag(t *testing.T) {
	var label BuildLabel
	assert.NoError(t, label.UnmarshalFlag("//src/core:core"))
	assert.Equal(t, label.PackageName, "src/core")
	assert.Equal(t, label.Name, "core")
	// N.B. we can't test a failure here because it does a log.Fatalf
}

func TestUnmarshalText(t *testing.T) {
	var label BuildLabel
	assert.NoError(t, label.UnmarshalText([]byte("//src/core:core")))
	assert.Equal(t, label.PackageName, "src/core")
	assert.Equal(t, label.Name, "core")
	assert.Error(t, label.UnmarshalText([]byte(":blahblah:")))
}

func TestString(t *testing.T) {
	label := BuildLabel{PackageName: "src/core", Name: "core"}
	assert.Equal(t, "//src/core:core", label.String())
	label = BuildLabel{PackageName: "src/core", Name: "core", Arch: "test_x86"}
	assert.Equal(t, "//src/core:core [test_x86]", label.String())
}

func TestToArch(t *testing.T) {
	label1 := BuildLabel{PackageName: "src/core", Name: "core"}
	label2 := label1.toArch("test_x86")
	assert.Equal(t, "", label1.Arch)
	assert.Equal(t, "test_x86", label2.Arch)
}

func TestNoArch(t *testing.T) {
	label1 := BuildLabel{PackageName: "src/core", Name: "core", Arch: "test_x86"}
	label2 := label1.noArch()
	assert.Equal(t, "test_x86", label1.Arch)
	assert.Equal(t, "", label2.Arch)
}

func TestOsArch(t *testing.T) {
	label := BuildLabel{PackageName: "src/core", Name: "core", Arch: "test_x86"}
	os, arch := label.OsArch()
	assert.Equal(t, "test", os)
	assert.Equal(t, "x86", arch)
}

func TestOsArchUnknownOs(t *testing.T) {
	label := BuildLabel{PackageName: "src/core", Name: "core", Arch: "wibble"}
	os, arch := label.OsArch()
	assert.Equal(t, "", os)
	assert.Equal(t, "wibble", arch)
}

func TestPackageDir(t *testing.T) {
	label := NewBuildLabel("src/core", "core")
	assert.Equal(t, "src/core", label.PackageDir())
	label = NewBuildLabel("", "core")
	assert.Equal(t, ".", label.PackageDir())
}
