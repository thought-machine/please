package output

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
