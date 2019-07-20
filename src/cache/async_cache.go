package cache

import (
	"sync"

	"github.com/thought-machine/please/src/core"
)

// An asyncCache is a wrapper around a Cache interface that handles incoming
// store requests asynchronously and attempts to return immediately.
// The requests are handled on an internal queue, if that fills up then
// incoming requests will start to block again until it empties.
// Retrieval requests are still handled synchronously.
type asyncCache struct {
	requests  chan cacheRequest
	realCache core.Cache
	wg        sync.WaitGroup
}

// A cacheRequest models an incoming cache request on our queue.
type cacheRequest struct {
	target   *core.BuildTarget
	metadata *core.BuildMetadata
	key      []byte
	files    []string
	file     string
}

func newAsyncCache(realCache core.Cache, config *core.Configuration) core.Cache {
	c := &asyncCache{
		requests:  make(chan cacheRequest),
		realCache: realCache,
	}
	c.wg.Add(config.Cache.Workers)
	for i := 0; i < config.Cache.Workers; i++ {
		go c.run()
	}
	return c
}

func (c *asyncCache) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
	c.requests <- cacheRequest{
		target:   target,
		metadata: metadata,
		key:      key,
		files:    files,
	}
}

func (c *asyncCache) Retrieve(target *core.BuildTarget, key []byte) *core.BuildMetadata {
	return c.realCache.Retrieve(target, key)
}

func (c *asyncCache) Clean(target *core.BuildTarget) {
	c.realCache.Clean(target)
}

func (c *asyncCache) CleanAll() {
	c.realCache.CleanAll()
}

func (c *asyncCache) Shutdown() {
	log.Info("Shutting down cache workers...")
	close(c.requests)
	c.wg.Wait()
	log.Debug("Shut down all cache workers")
}

// run implements the actual async logic.
func (c *asyncCache) run() {
	for r := range c.requests {
		c.realCache.Store(r.target, r.key, r.metadata, r.files)
	}
	c.wg.Done()
}
