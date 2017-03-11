package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestByteSize(t *testing.T) {
	opts := struct {
		Size ByteSize `short:"b"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test", "-b=15M"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, 15000000, opts.Size)
}

func TestDuration(t *testing.T) {
	opts := struct {
		D Duration `short:"d"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test", "-d=3h"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, 3*time.Hour, opts.D)

	_, extraArgs, err = ParseFlags("test", &opts, []string{"test", "-d=3"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, 3*time.Second, opts.D)
}

func TestDurationDefault(t *testing.T) {
	opts := struct {
		D Duration `short:"d" default:"3h"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, 3*time.Hour, opts.D)
}

func TestURL(t *testing.T) {
	opts := struct {
		U URL `short:"u"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test", "-u=https://localhost:8080"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extraArgs))
	assert.EqualValues(t, "https://localhost:8080", opts.U)
}

func TestURLDefault(t *testing.T) {
	opts := struct {
		U URL `short:"u" default:"https://localhost:8080"`
	}{}
	_, extraArgs, err := ParseFlags("test", &opts, []string{"test"})
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

func TestGetUsageTag(t *testing.T) {
	opts := struct {
		Usage string `usage:"Test usage"`
	}{}
	assert.Equal(t, "Test usage", getUsage(&opts))
}

func TestGetUsageValue(t *testing.T) {
	opts := struct {
		Usage string
	}{
		Usage: "Test usage",
	}
	assert.Equal(t, "Test usage", getUsage(&opts))
}

func TestGetUsageNoUsage(t *testing.T) {
	opts := struct{}{}
	assert.Equal(t, "", getUsage(&opts))
}
