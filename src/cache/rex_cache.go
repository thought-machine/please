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

func (rc *rexCache) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
	log.Debug("Storing %s in remote cache...", target.Label)
	if err := rc.client.Store(target, key, metadata, files); err != nil {
		log.Warning("Error storing artifacts in remote cache: %s", err)
	}
}

func (rc *rexCache) Retrieve(target *core.BuildTarget, key []byte) *core.BuildMetadata {
	metadata, err := rc.client.Retrieve(target, key)
	if err != nil {
		if remote.IsNotFound(err) {
			log.Debug("Artifacts for %s [key %s] don't exist in remote cache", target.Label, hex.EncodeToString(key))
		} else {
			log.Warning("Error retrieving artifacts for %s from remote cache: %s", target.Label, err)
		}
		return nil
	}
	return metadata
}

func (rc *rexCache) Clean(target *core.BuildTarget) {
	// There is no API for this, so we just don't do it.
}

func (rc *rexCache) CleanAll() {
	// Similarly here.
}

func (rc *rexCache) Shutdown() {
}
