package fs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameFile(t *testing.T) {
	err := os.WriteFile("issamefile1.txt", []byte("hello"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile("issamefile2.txt", []byte("hello"), 0644)
	assert.NoError(t, err)
	err = os.Link("issamefile1.txt", "issamefile3.txt")
	assert.NoError(t, err)
	assert.True(t, IsSameFile("issamefile1.txt", "issamefile3.txt"))
	assert.False(t, IsSameFile("issamefile1.txt", "issamefile2.txt"))
	assert.False(t, IsSameFile("issamefile1.txt", "doesntexist.txt"))
}

func TestEnsureDir(t *testing.T) {
	err := os.WriteFile("ensure_dir", []byte("hello"), 0644)
	assert.NoError(t, err)
	err = EnsureDir("ensure_dir/filename")
	assert.NoError(t, err)
}

func TestOpenDirFile(t *testing.T) {
	_, err := os.OpenFile("dir/file", os.O_RDWR|os.O_CREATE, 0644)
	assert.Error(t, err)

	file, err := OpenDirFile("dir/file", os.O_RDWR|os.O_CREATE, 0644)
	assert.IsType(t, &os.File{}, file)
	assert.NoError(t, err)
}

func TestIsPackage(t *testing.T) {
	isPackage := IsPackage([]string{"TEST_BUILD"}, "src/fs/test_data/test_subfolder1")
	assert.False(t, isPackage)

	isPackage = IsPackage([]string{"TEST_BUILD"}, "src/fs/test_data/test_subfolder4")
	assert.True(t, isPackage)
}
