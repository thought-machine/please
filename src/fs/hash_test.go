package fs

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha1.New, "")
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false, false)
	b2, err2 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}

func TestCopyHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha1.New, "")
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false, false)
	h.CopyHash("src/fs/test_data/test_subfolder1/a.txt", "doesnt_exist.txt")
	b2, err2 := h.Hash("doesnt_exist.txt", false, false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}

func TestSetHash(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha1.New, "")
	b1, err1 := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false, false)
	h.SetHash("doesnt_exist.txt", b1)
	b2, err2 := h.Hash("doesnt_exist.txt", false, false, false)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.EqualValues(t, b1, b2)
}

func TestHashConstructorSHA1(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha1.New, "")
	b, err := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false, false)
	assert.NoError(t, err)
	assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", hex.EncodeToString(b))
}

func TestHashConstructorSHA256(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha256.New, "_256")
	b, err := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false, false)
	assert.NoError(t, err)
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hex.EncodeToString(b))
}

func TestHashLastModified(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	h := NewPathHasher(wd, true, sha256.New, "_256")
	path := "src/fs/test_data/test_subfolder1/a.txt"
	modTime := time.Now().UTC()
	err = os.Chtimes(path, modTime, modTime)
	require.NoError(t, err)

	sha256Hash := sha256.New()
	sha256Hash.Write([]byte(modTime.Format(time.DateTime)))
	expected := sha256Hash.Sum(nil)

	b, err := h.Hash("src/fs/test_data/test_subfolder1/a.txt", false, false, true)
	assert.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(expected), hex.EncodeToString(b))
}
