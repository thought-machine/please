package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteSize(t *testing.T) {
	opts := struct {
		Size ByteSize `short:"b"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test", "-b=15M"}, 0, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, 15000000, opts.Size)
}

func TestURL(t *testing.T) {
	opts := struct {
		U URL `short:"u"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test", "-u=https://localhost:8080"}, 0, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, "https://localhost:8080", opts.U)
}

func TestURLDefault(t *testing.T) {
	opts := struct {
		U URL `short:"u" default:"https://localhost:8080"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test"}, 0, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, "https://localhost:8080", opts.U)
}

func TestVersion(t *testing.T) {
	v := Version{}
	assert.NoError(t, v.UnmarshalFlag("3.2.1"))
	assert.EqualValues(t, 3, v.Major)
	assert.EqualValues(t, 2, v.Minor)
	assert.EqualValues(t, 1, v.Patch)
	assert.False(t, v.IsGTE)
	assert.NoError(t, v.UnmarshalFlag(">=3.2.1"))
	assert.EqualValues(t, 3, v.Major)
	assert.EqualValues(t, 2, v.Minor)
	assert.EqualValues(t, 1, v.Patch)
	assert.True(t, v.IsGTE)
	assert.NoError(t, v.UnmarshalFlag(">= 3.2.1"))
	assert.EqualValues(t, 3, v.Major)
	assert.EqualValues(t, 2, v.Minor)
	assert.EqualValues(t, 1, v.Patch)
	assert.True(t, v.IsGTE)
	assert.Error(t, v.UnmarshalFlag("thirty-five ham and cheese sandwiches"))
}

func TestVersionString(t *testing.T) {
	v := Version{}
	v.UnmarshalFlag("3.2.1")
	assert.Equal(t, "3.2.1", v.String())
	v.UnmarshalFlag(">=3.2.1")
	assert.Equal(t, ">=3.2.1", v.String())
}

func TestArch(t *testing.T) {
	a := Arch{}
	assert.NoError(t, a.UnmarshalFlag("linux_amd64"))
	assert.Equal(t, "linux", a.OS)
	assert.Equal(t, "amd64", a.Arch)
	assert.Equal(t, "linux_amd64", a.String())
	assert.Error(t, a.UnmarshalFlag("wibble"))
	assert.Error(t, a.UnmarshalFlag("not/an_arch"))
}

func TestXOS(t *testing.T) {
	a := NewArch("darwin", "amd64")
	assert.Equal(t, "osx", a.XOS())
	a = NewArch("linux", "amd64")
	assert.Equal(t, "linux", a.XOS())
}

func TestXArch(t *testing.T) {
	a := NewArch("darwin", "amd64")
	assert.Equal(t, "x86_64", a.XArch())
	a = NewArch("linux", "x86")
	assert.Equal(t, "x86_32", a.XArch())
	a = NewArch("linux", "arm")
	assert.Equal(t, "arm", a.XArch())
}

func TestGoArch(t *testing.T) {
	a := NewArch("darwin", "amd64")
	assert.Equal(t, "amd64", a.GoArch())
	a = NewArch("linux", "x86")
	assert.Equal(t, "386", a.GoArch())
}
