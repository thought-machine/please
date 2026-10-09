package cache

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

var hash = []byte("12345678901234567890")
var b64Hash = base64.URLEncoding.EncodeToString(hash)

func writeFile(filename string, size int) {
	contents := bytes.Repeat([]byte{'p', 'l', 'z'}, size) // so this is three times the size...
	if err := os.MkdirAll(filepath.Dir(filename), core.DirPermissions); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filename, contents, 0644); err != nil {
		panic(err)
	}
}

func cachePath(target *core.BuildTarget, compress bool) string {
	if compress {
		return filepath.Join(".plz-cache-"+target.Label.PackageName, target.Label.PackageName, target.Label.Name, b64Hash+".tar.gz")
	}
	return filepath.Join(".plz-cache-"+target.Label.PackageName, target.Label.PackageName, target.Label.Name, b64Hash, target.Outputs(nil)[0])
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
	target := makeTarget2("//test1:target1", 20)
	cache.Store(target, hash, target.Outputs(nil))
	// Should now exist in cache at this path
	assert.True(t, inCache(target))
	assert.NotNil(t, cache.Retrieve(target, hash, target.Outputs(nil)))
	// Should be able to store it again without problems
	cache.Store(target, hash, target.Outputs(nil))
	assert.True(t, inCache(target))
	assert.NotNil(t, cache.Retrieve(target, hash, target.Outputs(nil)))
}

func TestCleanNoop(t *testing.T) {
	cache := makeCache(".plz-cache-test2", false)
	target1 := makeTarget2("//test2:target1", 2000)
	writeFile(cachePath(target1, false), 2000)
	target2 := makeTarget2("//test2:target2", 2000)
	writeFile(cachePath(target2, false), 2000)
	// Doesn't clean anything this time because the high water mark is sufficiently high
	totalSize := cache.clean(20000, 1000)
	assert.EqualValues(t, 12000, totalSize)
	assert.True(t, inCache(target1))
	assert.True(t, inCache(target2))
}

func TestCleanNoop2(t *testing.T) {
	cache := makeCache(".plz-cache-test3", false)
	target1 := makeTarget2("//test3:target1", 2000)
	cache.Store(target1, hash, target1.Outputs(nil))
	assert.True(t, inCache(target1))
	target2 := makeTarget2("//test3:target2", 2000)
	cache.Store(target2, hash, target2.Outputs(nil))
	assert.True(t, inCache(target2))
	// Doesn't clean anything this time; both targets have just been built, and are hard linked
	// into plz-out so they don't count towards the size anyway.
	totalSize := cache.clean(1, 0)
	assert.EqualValues(t, 0, totalSize)
	assert.True(t, inCache(target1))
	assert.True(t, inCache(target2))
}

func TestCleanForReal(t *testing.T) {
	cache := makeCache(".plz-cache-test4", false)
	target1 := makeTarget2("//test4:target1", 2000)
	cache.Store(target1, hash, target1.Outputs(nil))
	assert.True(t, inCache(target1))
	target2 := makeTarget2("//test4:target2", 2000)
	writeFile(cachePath(target2, false), 2000)
	assert.True(t, inCache(target2))
	// This time it should clean target2, because target1 has just been stored
	totalSize := cache.clean(5000, 1000)
	assert.EqualValues(t, 0, totalSize)
	assert.True(t, inCache(target1))
	assert.False(t, inCache(target2))
}

func TestCleanForReal2(t *testing.T) {
	cache := makeCache(".plz-cache-test5", false)
	target1 := makeTarget2("//test5:target1", 2000)
	writeFile(cachePath(target1, false), 2000)
	assert.True(t, inCache(target1))
	target2 := makeTarget2("//test5:target2", 2000)
	cache.Store(target2, hash, target2.Outputs(nil))
	assert.True(t, inCache(target2))
	// This time it should clean target1, because target2 has just been stored
	totalSize := cache.clean(5000, 1000)
	assert.EqualValues(t, 0, totalSize)
	assert.False(t, inCache(target1))
	assert.True(t, inCache(target2))
}

func TestCleanSkipsLinkedEntries(t *testing.T) {
	makeCache(".plz-cache-test8", false).Store(makeTarget2("//test8:target1", 2000), hash, []string{"test.go"})
	target1 := makeTarget2("//test8:target1", 2000)
	cache := makeCache(".plz-cache-test8", false) // A new instance doesn't know what the previous one stored.
	// Hardlink the file in again, as a second build would do on retrieving it.
	out := filepath.Join("plz-out/gen", target1.Label.PackageName, "test.go")
	assert.NoError(t, os.Remove(out))
	assert.NoError(t, os.Link(cachePath(target1, false), out))
	// The entry is still linked into plz-out so cleaning it won't save anything.
	assert.EqualValues(t, 0, cache.clean(1, 0))
	assert.True(t, inCache(target1))
	// Once that link is gone (e.g. the repo is deleted), it does count and can be cleaned.
	assert.NoError(t, os.Remove(out))
	assert.EqualValues(t, 0, cache.clean(1000, 0))
	assert.False(t, inCache(target1))
}

func TestCleanSharedWithinCache(t *testing.T) {
	cache := makeCache(".plz-cache-test9", false)
	target1 := makeTarget2("//test9:target1", 2000)
	writeFile(cachePath(target1, false), 2000)
	target2 := makeTarget2("//test9:target2", 2000)
	assert.NoError(t, os.MkdirAll(filepath.Dir(cachePath(target2, false)), core.DirPermissions))
	assert.NoError(t, os.Link(cachePath(target1, false), cachePath(target2, false)))
	setMtime(t, cachePath(target1, false), 2*time.Hour)
	// The file is only linked within the cache, so it counts but only once.
	assert.EqualValues(t, 6000, cache.clean(10000, 0))
	// Removing the first entry doesn't free anything, so it has to remove the second too.
	assert.EqualValues(t, 0, cache.clean(5000, 1000))
	assert.False(t, inCache(target1))
	assert.False(t, inCache(target2))
}

func TestCleanLeastRecentlyUsed(t *testing.T) {
	cache := makeCache(".plz-cache-test10", false)
	target1 := makeTarget2("//test10:target1", 2000)
	writeFile(cachePath(target1, false), 2000)
	setMtime(t, cachePath(target1, false), time.Hour)
	target2 := makeTarget2("//test10:target2", 2000)
	writeFile(cachePath(target2, false), 2000)
	setMtime(t, cachePath(target2, false), 3*time.Hour)
	target3 := makeTarget2("//test10:target3", 2000)
	writeFile(cachePath(target3, false), 2000)
	setMtime(t, cachePath(target3, false), 2*time.Hour)
	// Should remove the least recently used entry first, which is target2.
	assert.EqualValues(t, 12000, cache.clean(15000, 13000))
	assert.True(t, inCache(target1))
	assert.False(t, inCache(target2))
	assert.True(t, inCache(target3))
}

func TestRetrieveUpdatesMtime(t *testing.T) {
	cache := makeCache(".plz-cache-test11", false)
	target := makeTarget2("//test11:target1", 20)
	cache.Store(target, hash, target.Outputs(nil))
	entry := filepath.Dir(cachePath(target, false))
	// Not updated if it was recently modified
	setMtime(t, cachePath(target, false), 30*time.Minute)
	assert.True(t, cache.Retrieve(target, hash, target.Outputs(nil)))
	assert.True(t, time.Since(getMtime(t, entry)) > 20*time.Minute)
	// But it is updated if it's older
	setMtime(t, cachePath(target, false), 2*time.Hour)
	assert.True(t, cache.Retrieve(target, hash, target.Outputs(nil)))
	assert.True(t, time.Since(getMtime(t, entry)) < time.Minute)
}

func TestStoreAndRetrieveCompressed(t *testing.T) {
	cache := makeCache(".plz-cache-test6", true)
	target := makeTarget2("//test6:target6", 20)
	cache.Store(target, hash, target.Outputs(nil))
	// Should now exist in cache at this path
	assert.True(t, inCompressedCache(target))
	assert.NotNil(t, cache.Retrieve(target, hash, target.Outputs(nil)))
	// Should be able to store it again without problems
	cache.Store(target, hash, target.Outputs(nil))
	assert.True(t, inCompressedCache(target))
	assert.NotNil(t, cache.Retrieve(target, hash, target.Outputs(nil)))
}

func TestCleanCompressed(t *testing.T) {
	cache := makeCache(".plz-cache-test7", true)
	target1 := makeTarget2("//test7:target1", 2000)
	writeFile(cachePath(target1, true), 2000)
	assert.True(t, inCompressedCache(target1))
	target2 := makeTarget2("//test7:target2", 2000)
	cache.Store(target2, hash, target2.Outputs(nil))
	assert.True(t, inCompressedCache(target2))
	// Don't want to assert the size here since it depends on how well gzip compresses.
	// It's a bit hard to know exactly what the sizes here should be too but we'll guess
	// and assume it won't change dramatically.
	cache.clean(3000, 1000)
	assert.False(t, inCompressedCache(target1))
	assert.True(t, inCompressedCache(target2))
}

// setMtime sets the modification time of the cache entry containing the given file to the given time ago.
func setMtime(t *testing.T, filename string, ago time.Duration) {
	t.Helper()
	then := time.Now().Add(-ago)
	assert.NoError(t, os.Chtimes(filepath.Dir(filename), then, then))
}

func getMtime(t *testing.T, filename string) time.Time {
	t.Helper()
	info, err := os.Stat(filename)
	assert.NoError(t, err)
	return info.ModTime()
}

func makeCache(dir string, compress bool) *dirCache {
	config := core.DefaultConfiguration()
	config.Cache.Dir = dir
	config.Cache.DirClean = false // We will do this explicitly
	config.Cache.DirCompress = compress
	return newDirCache(config)
}

func makeTarget2(label string, size int) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.AddOutput("test.go")
	writeFile(filepath.Join("plz-out/gen", target.Label.PackageName, "test.go"), size)
	return target
}
