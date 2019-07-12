// Caching support for Please.

package cache

import (
	"github.com/thought-machine/please/src/core"
	"net/http"
	"sync"

	"gopkg.in/op/go-logging.v1"
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
	if state.Config.Remote.URL != "" {
		mplex.caches = append(mplex.caches, newRemoteCache(state))
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
		res, err := http.Get(state.Config.Cache.HTTPURL.String() + "/ping")
		if err == nil && res.StatusCode == 200 {
			mplex.caches = append(mplex.caches, newHTTPCache(state.Config))
		} else {
			log.Warning("Http cache server could not be reached: %s.\nSkipping http caching...", err)
		}
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

func (mplex cacheMultiplexer) Store(target *core.BuildTarget, key []byte, files ...string) {
	mplex.storeUntil(target, key, files, len(mplex.caches))
}

// storeUntil stores artifacts into higher priority caches than the given one.
// Used after artifact retrieval to ensure we have them in eg. the directory cache after
// downloading from the RPC cache.
// This is a little inefficient since we could write the file to plz-out then copy it to the dir cache,
// but it's hard to fix that without breaking the cache abstraction.
func (mplex cacheMultiplexer) storeUntil(target *core.BuildTarget, key []byte, files []string, stopAt int) {
	// Attempt to store on all caches simultaneously.
	var wg sync.WaitGroup
	for i, cache := range mplex.caches {
		if i == stopAt {
			break
		}
		wg.Add(1)
		go func(cache core.Cache) {
			cache.Store(target, key, files...)
			wg.Done()
		}(cache)
	}
	wg.Wait()
}

func (mplex cacheMultiplexer) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	mplex.storeExtraUntil(target, key, file, len(mplex.caches))
}

// storeExtraUntil is similar to storeUntil but stores a single file.
func (mplex cacheMultiplexer) storeExtraUntil(target *core.BuildTarget, key []byte, file string, stopAt int) {
	// Attempt to store on all caches simultaneously.
	var wg sync.WaitGroup
	for i, cache := range mplex.caches {
		if i == stopAt {
			break
		}
		wg.Add(1)
		go func(cache core.Cache) {
			cache.StoreExtra(target, key, file)
			wg.Done()
		}(cache)
	}
	wg.Wait()
}

func (mplex cacheMultiplexer) Retrieve(target *core.BuildTarget, key []byte) bool {
	// Retrieve from caches sequentially; if we did them simultaneously we could
	// easily write the same file from two goroutines at once.
	for i, cache := range mplex.caches {
		if cache.Retrieve(target, key) {
			// Store this into other caches
			mplex.storeUntil(target, key, nil, i)
			return true
		}
	}
	return false
}

func (mplex cacheMultiplexer) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	// Retrieve from caches sequentially; if we did them simultaneously we could
	// easily write the same file from two goroutines at once.
	for i, cache := range mplex.caches {
		if cache.RetrieveExtra(target, key, file) {
			// Store this into other caches
			mplex.storeExtraUntil(target, key, file, i)
			return true
		}
	}
	return false
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

// Returns all cacheable artifacts from this target.
func cacheArtifacts(target *core.BuildTarget, files ...string) []string {
	return append(target.Outputs(), files...)
}
