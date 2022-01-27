package fs

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha1.New, "")
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	b2, err2 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}

func TestMoveHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha1.New, "")
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
	h := NewPathHasher(wd, true, sha1.New, "")
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	h.SetHash("doesnt_exist.txt", b1)
	b2, err2 := h.Hash("doesnt_exist.txt", false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}

func TestHashConstructorSHA1(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha1.New, "")
	b, err := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	assert.NoError(t, err)
	assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", hex.EncodeToString(b))
}
func TestHashConstructorSHA256(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha256.New, "_256")
	b, err := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false)
	assert.NoError(t, err)
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hex.EncodeToString(b))
}
