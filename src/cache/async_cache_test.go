package cache

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"core"
)

func TestStore(t *testing.T) {
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_store")
	aCache.Store(target, nil)
	aCache.Shutdown()
	assert.False(t, mCache.inFlight[target])
	assert.True(t, mCache.completed[target])
}

func TestStoreExtra(t *testing.T) {
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_store_extra")
	aCache.StoreExtra(target, nil, "some_other_file")
	aCache.Shutdown()
	assert.False(t, mCache.inFlight[target])
	assert.True(t, mCache.completed[target])
}

func TestRetrieve(t *testing.T) {
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_retrieve")
	aCache.Retrieve(target, nil)
	aCache.Shutdown()
	assert.False(t, mCache.inFlight[target])
	assert.True(t, mCache.completed[target])
}

func TestRetrieveExtra(t *testing.T) {
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_retrieve_extra")
	aCache.RetrieveExtra(target, nil, "some_other_file")
	aCache.Shutdown()
	assert.False(t, mCache.inFlight[target])
	assert.True(t, mCache.completed[target])
}

func TestClean(t *testing.T) {
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_clean")
	aCache.Clean(target)
	aCache.Shutdown()
	assert.False(t, mCache.inFlight[target])
	assert.True(t, mCache.completed[target])
}

func TestConcurrentStores(t *testing.T) {
	// The cache shouldn't run multiple concurrent stores for the same target.
	// Our mock cache will panic if it detects that, so here we just throw enough
	// concurrent requests at it to try to make sure that we're likely to exercise that.
	// It's pretty hard to really guarantee that it does happen though.
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_concurrent")
	expected := []string{}
	for i := 0; i < 20; i++ {
		s := fmt.Sprintf("file%02d", i)
		aCache.StoreExtra(target, nil, s)
		expected = append(expected, s)
	}
	aCache.Shutdown()
	assert.False(t, mCache.inFlight[target])
	assert.True(t, mCache.completed[target])
	stored := mCache.stored[target]
	sort.Strings(stored)
	assert.Equal(t, expected, stored)
}

// Fake cache implementation to ensure our async cache behaves itself.
type mockCache struct {
	sync.Mutex
	inFlight  map[*core.BuildTarget]bool
	completed map[*core.BuildTarget]bool
	stored    map[*core.BuildTarget][]string
}

func (c *mockCache) Store(target *core.BuildTarget, key []byte) {
	c.StoreExtra(target, key, "")
}

func (c *mockCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	c.Lock()
	if c.inFlight[target] {
		panic("Concurrent store on " + target.Label.String())
	}
	c.inFlight[target] = true
	c.Unlock()
	time.Sleep(10 * time.Millisecond) // Fake a small delay to mimic the real thing
	c.Lock()
	c.inFlight[target] = false
	c.completed[target] = true
	c.stored[target] = append(c.stored[target], file)
	c.Unlock()
}

func (c *mockCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	c.Lock()
	c.completed[target] = true
	c.Unlock()
	return false
}

func (c *mockCache) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	return c.Retrieve(target, key)
}

func (c *mockCache) Clean(target *core.BuildTarget) {
	c.Retrieve(target, nil)
}

func (*mockCache) Shutdown() {}

func makeTarget(label string) *core.BuildTarget {
	return core.NewBuildTarget(core.ParseBuildLabel(label, ""))
}

func makeCaches() (mockCache, core.Cache) {
	mCache := mockCache{
		inFlight:  make(map[*core.BuildTarget]bool),
		completed: make(map[*core.BuildTarget]bool),
		stored:    make(map[*core.BuildTarget][]string),
	}
	return mCache, newAsyncCache(&mCache, core.DefaultConfiguration())
}
