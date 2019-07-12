// +build !bootstrap

// Cache based on the Google remote execution API.
// This will likely supersede the RPC cache at some point in the future.

package cache

import (
	"encoding/hex"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/remote"
)

type rexCache struct {
	client *remote.Client
}

func newRemoteCache(state *core.BuildState) *rexCache {
	return &rexCache{client: remote.Get(state)}
}

func (rc *rexCache) Store(target *core.BuildTarget, key []byte, files ...string) {
	log.Debug("Storing %s in remote cache...", target.Label)
	if err := rc.client.Store(target, key, cacheArtifacts(target, files...)); err != nil {
		log.Warning("Error storing artifacts in remote cache: %s", err)
	}
}

func (c *rexCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	log.Debug("Storing %s: %s in remote cache...", target.Label, file)
}

func (rc *rexCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	if err := rc.client.Retrieve(target, key); err != nil {
		if remote.IsNotFound(err) {
			log.Debug("Artifacts for %s [key %s] don't exist in remote cache", target.Label, hex.EncodeToString(key))
		} else {
			log.Warning("Error retrieving artifacts for %s from remote cache: %s", target.Label, err)
		}
		return false
	}
	return true
}

func (rc *rexCache) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	return false
}

func (rc *rexCache) Clean(target *core.BuildTarget) {
	// There is no API for this, so we just don't do it.
}

func (rc *rexCache) CleanAll() {
	// Similarly here.
}

func (rc *rexCache) Shutdown() {
}
