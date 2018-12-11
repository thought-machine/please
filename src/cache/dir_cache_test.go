package cache

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

var hash = []byte("12345678901234567890")
var b64Hash = base64.URLEncoding.EncodeToString(hash)

func writeFile(filename string, size int) {
	contents := bytes.Repeat([]byte{'p', 'l', 'z'}, size) // so this is three times the size...
	if err := os.MkdirAll(path.Dir(filename), core.DirPermissions); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, contents, 0644); err != nil {
		panic(err)
	}
}

func cachePath(target *core.BuildTarget, compress bool) string {
	if compress {
		return path.Join(".plz-cache-"+target.Label.PackageName, target.Label.PackageName, target.Label.Name, b64Hash+".tar.gz")
	}
	return path.Join(".plz-cache-"+target.Label.PackageName, target.Label.PackageName, target.Label.Name, b64Hash, target.Outputs()[0])
}

func inCache(target *core.BuildTarget) bool {
	dest := cachePath(target, false)
	log.Debug("Checking for %s", dest)
	return core.PathExists(dest)
}

func inCompressedCache(target *core.BuildTarget) bool {
	dest := cachePath(target, true)
	log.Debug("Checking for %s", dest)
	return core.PathExists(dest)
}

func TestStoreAndRetrieve(t *testing.T) {
	cache := makeCache(".plz-cache-test1", false)
	target := makeTarget("//test1:target1", 20)
	cache.Store(target, hash)
	// Should now exist in cache at this path
	assert.True(t, inCache(target))
	assert.True(t, cache.Retrieve(target, hash))
	// Should be able to store it again without problems
	cache.Store(target, hash)
	assert.True(t, inCache(target))
	assert.True(t, cache.Retrieve(target, hash))
}

func TestCleanNoop(t *testing.T) {
	cache := makeCache(".plz-cache-test2", false)
	target1 := makeTarget("//test2:target1", 2000)
	cache.Store(target1, hash)
	assert.True(t, inCache(target1))
	target2 := makeTarget("//test2:target2", 2000)
	cache.Store(target2, hash)
	assert.True(t, inCache(target2))
	// Doesn't clean anything this time because the high water mark is sufficiently high
	totalSize := cache.clean(20000, 1000)
	assert.EqualValues(t, 12000, totalSize)
	assert.True(t, inCache(target1))
	assert.True(t, inCache(target2))
}

func TestCleanNoop2(t *testing.T) {
	cache := makeCache(".plz-cache-test3", false)
	target1 := makeTarget("//test3:target1", 2000)
	cache.Store(target1, hash)
	assert.True(t, inCache(target1))
	target2 := makeTarget("//test3:target2", 2000)
	cache.Store(target2, hash)
	assert.True(t, inCache(target2))
	// Doesn't clean anything this time, the high water mark is lower but both targets have
	// just been built.
	totalSize := cache.clean(10000, 1000)
	assert.EqualValues(t, 12000, totalSize)
	assert.True(t, inCache(target1))
	assert.True(t, inCache(target2))
}

func TestCleanForReal(t *testing.T) {
	cache := makeCache(".plz-cache-test4", false)
	target1 := makeTarget("//test4:target1", 2000)
	cache.Store(target1, hash)
	assert.True(t, inCache(target1))
	target2 := makeTarget("//test4:target2", 2000)
	writeFile(cachePath(target2, false), 2000)
	assert.True(t, inCache(target2))
	// This time it should clean target2, because target1 has just been stored
	totalSize := cache.clean(10000, 1000)
	assert.EqualValues(t, 6000, totalSize)
	assert.True(t, inCache(target1))
	assert.False(t, inCache(target2))
}

func TestCleanForReal2(t *testing.T) {
	cache := makeCache(".plz-cache-test5", false)
	target1 := makeTarget("//test5:target1", 2000)
	writeFile(cachePath(target1, false), 2000)
	assert.True(t, inCache(target1))
	target2 := makeTarget("//test5:target2", 2000)
	cache.Store(target2, hash)
	assert.True(t, inCache(target2))
	// This time it should clean target1, because target2 has just been stored
	totalSize := cache.clean(10000, 1000)
	assert.EqualValues(t, 6000, totalSize)
	assert.False(t, inCache(target1))
	assert.True(t, inCache(target2))
}

func TestStoreAndRetrieveCompressed(t *testing.T) {
	cache := makeCache(".plz-cache-test6", true)
	target := makeTarget("//test6:target6", 20)
	cache.Store(target, hash)
	// Should now exist in cache at this path
	assert.True(t, inCompressedCache(target))
	assert.True(t, cache.Retrieve(target, hash))
	// Should be able to store it again without problems
	cache.Store(target, hash)
	assert.True(t, inCompressedCache(target))
	assert.True(t, cache.Retrieve(target, hash))
}

func TestCleanCompressed(t *testing.T) {
	cache := makeCache(".plz-cache-test7", true)
	target1 := makeTarget("//test7:target1", 2000)
	writeFile(cachePath(target1, true), 2000)
	assert.True(t, inCompressedCache(target1))
	target2 := makeTarget("//test7:target2", 2000)
	cache.Store(target2, hash)
	assert.True(t, inCompressedCache(target2))
	// Don't want to assert the size here since it depends on how well gzip compresses.
	// It's a bit hard to know exactly what the sizes here should be too but we'll guess
	// and assume it won't change dramatically.
	cache.clean(3000, 1000)
	assert.False(t, inCompressedCache(target1))
	assert.True(t, inCompressedCache(target2))
}

func makeCache(dir string, compress bool) *dirCache {
	config := core.DefaultConfiguration()
	config.Cache.Dir = dir
	config.Cache.DirClean = false // We will do this explicitly
	config.Cache.DirCompress = compress
	return newDirCache(config)
}

func makeTarget(label string, size int) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.AddOutput("test.go")
	writeFile(path.Join("plz-out/gen", target.Label.PackageName, "test.go"), size)
	return target
}
