package cache

import (
	"sync"

	"core"
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
	mutex     sync.Mutex
	// queuedRequests handles the awkward case of storing multiple things for one
	// build target, which isn't necessarily safe to do in parallel. This blocks
	// further requests from handling that target and queues them here until it's ready.
	queuedRequests map[*core.BuildTarget][]cacheRequest
}

// A cacheRequest models an incoming cache request on our queue.
type cacheRequest struct {
	target *core.BuildTarget
	key    []byte
	file   string
}

func newAsyncCache(realCache core.Cache, config *core.Configuration) core.Cache {
	c := &asyncCache{
		requests:       make(chan cacheRequest),
		realCache:      realCache,
		queuedRequests: make(map[*core.BuildTarget][]cacheRequest),
	}
	c.wg.Add(config.Cache.Workers)
	for i := 0; i < config.Cache.Workers; i++ {
		go c.run()
	}
	return c
}

func (c *asyncCache) Store(target *core.BuildTarget, key []byte) {
	c.requests <- cacheRequest{
		target: target,
		key:    key,
	}
}

func (c *asyncCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	c.requests <- cacheRequest{
		target: target,
		key:    key,
		file:   file,
	}
}

func (c *asyncCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	return c.realCache.Retrieve(target, key)
}

func (c *asyncCache) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	return c.realCache.RetrieveExtra(target, key, file)
}

func (c *asyncCache) Clean(target *core.BuildTarget) {
	c.realCache.Clean(target)
}

func (c *asyncCache) Shutdown() {
	log.Notice("Shutting down cache workers...")
	close(c.requests)
	c.wg.Wait()
}

// run implements the actual async logic.
func (c *asyncCache) run() {
	for r := range c.requests {
		// Ensure only one goroutine handles each target at a time.
		c.mutex.Lock()
		q, present := c.queuedRequests[r.target]
		if present {
			c.queuedRequests[r.target] = append(q, r)
			c.mutex.Unlock()
			continue
		}
		c.queuedRequests[r.target] = nil
		c.mutex.Unlock()
		c.runOne(r)
		// Now handle any queued requests that happened while we were dealing with it.
		c.mutex.Lock()
		q = c.queuedRequests[r.target]
		delete(c.queuedRequests, r.target)
		c.mutex.Unlock()
		for _, r := range q {
			c.runOne(r)
		}
	}
	log.Debug("Cache worker finished")
	c.wg.Done()
}

// runOne runs a single cache request.
func (c *asyncCache) runOne(r cacheRequest) {
	if r.file != "" {
		c.realCache.StoreExtra(r.target, r.key, r.file)
	} else {
		c.realCache.Store(r.target, r.key)
	}
}
