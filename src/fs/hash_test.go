package fs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true)
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	b2, err2 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}

func TestMoveHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true)
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	h.MoveHash("src/fs/test_data/test_subfolder1/a.txt", "doesnt_exist.txt", true)
	b2, err2 := h.Hash("doesnt_exist.txt", false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}

func TestSetHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true)
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	h.SetHash("doesnt_exist.txt", b1)
	b2, err2 := h.Hash("doesnt_exist.txt", false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}
