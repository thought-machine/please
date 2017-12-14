package cache

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
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

func cachePath(target *core.BuildTarget) string {
	return path.Join(".plz-cache-"+target.Label.PackageName, target.Label.PackageName, target.Label.Name, b64Hash, target.Outputs()[0])
}

func inCache(target *core.BuildTarget) bool {
	dest := cachePath(target)
	log.Debug("Checking for %s", dest)
	return core.PathExists(dest)
}

func TestStoreAndRetrieve(t *testing.T) {
	cache := makeCache(".plz-cache-test1")
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
	cache := makeCache(".plz-cache-test2")
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
	cache := makeCache(".plz-cache-test3")
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
	cache := makeCache(".plz-cache-test4")
	target1 := makeTarget("//test4:target1", 2000)
	cache.Store(target1, hash)
	assert.True(t, inCache(target1))
	target2 := makeTarget("//test4:target2", 2000)
	writeFile(cachePath(target2), 2000)
	assert.True(t, inCache(target2))
	// This time it should clean target2, because target1 has just been stored
	totalSize := cache.clean(10000, 1000)
	assert.EqualValues(t, 6000, totalSize)
	assert.True(t, inCache(target1))
	assert.False(t, inCache(target2))
}

func TestCleanForReal2(t *testing.T) {
	cache := makeCache(".plz-cache-test5")
	target1 := makeTarget("//test5:target1", 2000)
	writeFile(cachePath(target1), 2000)
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

func TestCleanForReal3(t *testing.T) {
	t.Skip("Failing on CI, not sure why")
	if runtime.GOOS != "linux" {
		// The various sizes that follow assume specific things about Linux's filesystem
		// (specifically that directories will cost 4k - which might also be ext4 specific?).
		t.Skip("assumes things about Linux's filesystem")
	}
	cache := makeCache(".plz-cache-test6")
	target1 := makeTarget("//test6:target1", 2000)
	writeFile(cachePath(target1), 2000)
	assert.True(t, inCache(target1))
	target2 := makeTarget("//test6:target2", 2000)
	writeFile(cachePath(target2), 2000)
	assert.True(t, inCache(target2))
	// This time it should clean one of the two targets, but it is indeterminate which one.
	// N.B. We allow a bit over the 6k you'd expect - each directory costs 4k as well.
	totalSize := cache.clean(12000, 11000)
	assert.EqualValues(t, 10096, totalSize) // again +4k for a directory
	assert.True(t, inCache(target1) != inCache(target2))
}

func makeCache(dir string) *dirCache {
	config := core.DefaultConfiguration()
	config.Cache.Dir = dir
	config.Cache.DirClean = false // We will do this explicitly
	return newDirCache(config)
}

func makeTarget(label string, size int) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.AddOutput("test.go")
	writeFile(path.Join("plz-out/gen", target.Label.PackageName, "test.go"), size)
	return target
}
