package tools

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPoint(t *testing.T) {
	assert.EqualValues(t, 0, HashPoint(0, 1))
	assert.EqualValues(t, 0, HashPoint(0, 5))
	assert.EqualValues(t, math.MaxUint32, HashPoint(1, 1))
	assert.EqualValues(t, math.MaxUint32, HashPoint(5, 5))
	assert.EqualValues(t, math.MaxUint32/2, HashPoint(1, 2))
}

func TestHash(t *testing.T) {
	assert.EqualValues(t, 0, Hash([]byte{0, 0, 0, 0}))
	// Little endian...
	assert.EqualValues(t, 1, Hash([]byte{1, 0, 0, 0}))
	// Bytes after the fourth are ignored
	assert.EqualValues(t, 1, Hash([]byte{1, 0, 0, 0, 15}))
	assert.EqualValues(t, math.MaxUint32, Hash([]byte{255, 255, 255, 255}))
}

func TestAlternateHash(t *testing.T) {
	// The alternate hash should move it halfway through the hash space.
	assert.EqualValues(t, 1<<31, AlternateHash([]byte{0, 0, 0, 0}))
	assert.EqualValues(t, 1+1<<31, AlternateHash([]byte{1, 0, 0, 0}))
	assert.EqualValues(t, 1<<31-1, AlternateHash([]byte{255, 255, 255, 255}))
}
