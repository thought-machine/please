// Tests for the core cache functionality
package server

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

var cache *Cache

const cachePath = "tools/cache/server/test_data"

func init() {
	cache = newCache(cachePath)
	core.NewBuildState(1, nil, 4, core.DefaultConfiguration())
}

func TestFilesToClean(t *testing.T) {
	c := newCache("test_files_to_clean")
	c.cachedFiles.Set("test/artifact/1", &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -2),
		readCount:    0,
		size:         1000,
	})
	c.cachedFiles.Set("test/artifact/2", &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -5),
		readCount:    0,
		size:         1000,
	})
	c.cachedFiles.Set("test/artifact/3", &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -1),
		readCount:    0,
		size:         1000,
	})
	c.totalSize = 3000

	paths := c.filesToClean(1700)
	assert.Equal(t, 2, len(paths))
}

func TestCleanOldFiles(t *testing.T) {
	c := newCache("test_clean_old_files")
	c.cachedFiles.Set("test/artifact/1", &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -2),
		readCount:    0,
		size:         1000,
	})
	c.cachedFiles.Set("test/artifact/2", &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -5),
		readCount:    0,
		size:         1000,
	})
	c.cachedFiles.Set("test/artifact/3", &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -1),
		readCount:    0,
		size:         1000,
	})
	c.totalSize = 3000
	assert.True(t, c.cleanOldFiles(72*time.Hour))
	assert.Equal(t, 2, c.cachedFiles.Count())
}

func TestRetrieve(t *testing.T) {
	artifact, err := cache.RetrieveArtifact("darwin_amd64/pack/label/hash/label.ext")
	assert.NoError(t, err)
	if artifact == nil {
		t.Error("Expected artifact and found nil.")
	}
}

func TestRetrieveError(t *testing.T) {
	artifact, err := cache.RetrieveArtifact(cachePath + "/darwin_amd64/somepack/somelabel/somehash/somelabel.ext")
	if artifact != nil {
		t.Error("Expected nil and found artifact.")
	}
	if err == nil {
		t.Error("Expected error and found nil.")
	}
}

func TestGlob(t *testing.T) {
	ret, err := cache.RetrieveArtifact("darwin_amd64/**/*.ext")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ret))
	assert.Equal(t, ret[0].File, "darwin_amd64/pack/label/hash/label.ext")
}

func TestStore(t *testing.T) {
	fileContent := "This is a newly created file."
	reader := strings.NewReader(fileContent)
	key, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)
	err = cache.StoreArtifact("/darwin_amd64/somepack/somelabel/somehash/somelabel.ext", key, "")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteArtifact(t *testing.T) {
	err := cache.DeleteArtifact("/linux_amd64/otherpack/label")
	assert.NoError(t, err)
	absPath, _ := filepath.Abs(cachePath + "/linux_amd64/otherpack/label")
	if _, err := os.Stat(absPath); err == nil {
		t.Errorf("%s was not removed from cache.", absPath)
	}
}

func TestDeleteAll(t *testing.T) {
	err := cache.DeleteAllArtifacts()
	assert.NoError(t, err)
	absPath, _ := filepath.Abs(cachePath)
	if files, _ := ioutil.ReadDir(absPath); len(files) != 0 {

		t.Error(files[0].Name())
		t.Error("The cache was not cleaned.")
	}
}
