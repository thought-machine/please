// Caching support for Please.

package cache

import (
	"sync"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
)

var log = logging.MustGetLogger("cache")

// NewCache is the factory function for creating a cache setup from the given config.
func NewCache(state *core.BuildState) core.Cache {
	c := newSyncCache(state, false)
	if state.Config.Cache.Workers > 0 {
		return newAsyncCache(c, state.Config)
	}
	return c
}

// newSyncCache creates a new cache, possibly multiplexing many underneath.
func newSyncCache(state *core.BuildState, remoteOnly bool) core.Cache {
	mplex := &cacheMultiplexer{}
	if state.Config.Cache.Dir != "" && !remoteOnly {
		mplex.caches = append(mplex.caches, newDirCache(state.Config))
	}
	if state.Config.Cache.RPCURL != "" {
		cache, err := newRPCCache(state.Config)
		if err == nil {
			mplex.caches = append(mplex.caches, cache)
		} else {
			log.Warning("RPC cache server could not be reached: %s", err)
		}
	}
	if state.Config.Cache.HTTPURL != "" {
		mplex.caches = append(mplex.caches, newHTTPCache(state.Config))
	}
	if len(mplex.caches) == 0 {
		return nil
	} else if len(mplex.caches) == 1 {
		return mplex.caches[0] // Skip the extra layer of indirection
	}
	return mplex
}

// A cacheMultiplexer multiplexes several caches into one.
// Used when we have several active (eg. http, dir).
type cacheMultiplexer struct {
	caches []core.Cache
}

func (mplex cacheMultiplexer) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
	mplex.storeUntil(target, key, metadata, files, len(mplex.caches))
}

// storeUntil stores artifacts into higher priority caches than the given one.
// Used after artifact retrieval to ensure we have them in eg. the directory cache after
// downloading from the RPC cache.
// This is a little inefficient since we could write the file to plz-out then copy it to the dir cache,
// but it's hard to fix that without breaking the cache abstraction.
func (mplex cacheMultiplexer) storeUntil(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string, stopAt int) {
	// Attempt to store on all caches simultaneously.
	var wg sync.WaitGroup
	for i, cache := range mplex.caches {
		if i == stopAt {
			break
		}
		wg.Add(1)
		go func(cache core.Cache) {
			cache.Store(target, key, metadata, files)
			wg.Done()
		}(cache)
	}
	wg.Wait()
}

func (mplex cacheMultiplexer) Retrieve(target *core.BuildTarget, key []byte, files []string) *core.BuildMetadata {
	// Retrieve from caches sequentially; if we did them simultaneously we could
	// easily write the same file from two goroutines at once.
	for i, cache := range mplex.caches {
		if metadata := cache.Retrieve(target, key, files); metadata != nil {
			// Store this into other caches
			mplex.storeUntil(target, key, metadata, files, i)
			return metadata
		}
	}
	return nil
}

func (mplex cacheMultiplexer) Clean(target *core.BuildTarget) {
	for _, cache := range mplex.caches {
		cache.Clean(target)
	}
}

func (mplex cacheMultiplexer) CleanAll() {
	for _, cache := range mplex.caches {
		cache.CleanAll()
	}
}

func (mplex cacheMultiplexer) Shutdown() {
	for _, cache := range mplex.caches {
		cache.Shutdown()
	}
}
