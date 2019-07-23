package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestStore(t *testing.T) {
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_store")
	aCache.Store(target, nil, &core.BuildMetadata{}, target.Outputs())
	aCache.Shutdown()
	assert.False(t, mCache.inFlight[target])
	assert.True(t, mCache.completed[target])
}

func TestRetrieve(t *testing.T) {
	mCache, aCache := makeCaches()
	target := makeTarget("//pkg1:test_retrieve")
	aCache.Retrieve(target, nil, target.Outputs())
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

func TestSimulateBuild(t *testing.T) {
	// Attempt to simulate what a normal build would do and confirm that the actions come
	// back out in the correct order.
	// This is a little obsolete now, it was ultimately solved by adding extra arguments to Store
	// instead of requiring extra calls to StoreExtra, but that means there isn't that much
	// left to exercise in this test any more.
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	mCache, aCache := makeCaches()
	for i := 0; i < n; i++ {
		go func(i int) {
			target := makeTarget(fmt.Sprintf("//test_pkg:target%03d", i))
			aCache.Store(target, nil, &core.BuildMetadata{}, []string{fmt.Sprintf("file%03d", i), fmt.Sprintf("file%03d_2", i)})
			wg.Done()
		}(i)
	}
	wg.Wait()
	aCache.Shutdown()
	assert.Equal(t, n, len(mCache.stored))
	for target, stored := range mCache.stored {
		assert.Equal(t, []string{
			"",
			"file" + target.Label.Name[len(target.Label.Name)-3:],
			"file" + target.Label.Name[len(target.Label.Name)-3:] + "_2",
		}, stored)
	}
}

// Fake cache implementation to ensure our async cache behaves itself.
type mockCache struct {
	sync.Mutex
	inFlight  map[*core.BuildTarget]bool
	completed map[*core.BuildTarget]bool
	stored    map[*core.BuildTarget][]string
}

func (c *mockCache) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
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
	c.stored[target] = append(c.stored[target], "")
	c.stored[target] = append(c.stored[target], files...)
	c.Unlock()
}

func (c *mockCache) Retrieve(target *core.BuildTarget, key []byte, files []string) *core.BuildMetadata {
	c.Lock()
	c.completed[target] = true
	c.Unlock()
	return nil
}

func (c *mockCache) Clean(target *core.BuildTarget) {
	c.Retrieve(target, nil, nil)
}

func (c *mockCache) CleanAll() {}

func (*mockCache) Shutdown() {}

func makeTarget(label string) *core.BuildTarget {
	return core.NewBuildTarget(core.ParseBuildLabel(label, ""))
}

func makeCaches() (*mockCache, core.Cache) {
	mCache := &mockCache{
		inFlight:  make(map[*core.BuildTarget]bool),
		completed: make(map[*core.BuildTarget]bool),
		stored:    make(map[*core.BuildTarget][]string),
	}
	config := core.DefaultConfiguration()
	config.Cache.Workers = 10
	return mCache, newAsyncCache(mCache, config)
}
