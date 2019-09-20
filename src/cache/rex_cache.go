// Cache based on the Google remote execution API.
// This will likely supersede the RPC cache at some point in the future.

package cache

import (
	"bytes"
	"encoding/hex"
	"path"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/remote"
)

type rexCache struct {
	client   core.RemoteClient
	readonly bool
}

func newRemoteCache(state *core.BuildState) *rexCache {
	return &rexCache{client: state.RemoteClient, readonly: state.Config.Remote.ReadOnly}
}

func (rc *rexCache) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
	if !rc.readonly {
		log.Debug("Storing %s in remote cache...", target.Label)
		if err := rc.client.Store(target, metadata, files); err != nil {
			log.Warning("Error storing artifacts in remote cache: %s", err)
		}
	}
}

func (rc *rexCache) Retrieve(target *core.BuildTarget, key []byte, files []string) *core.BuildMetadata {
	log.Debug("Retrieving %s from remote cache...", target.Label)
	metadata, err := rc.client.Retrieve(target)
	if err != nil {
		if remote.IsNotFound(err) {
			log.Debug("Artifacts for %s [key %s] don't exist in remote cache", target.Label, hex.EncodeToString(key))
		} else {
			log.Warning("Error retrieving artifacts for %s from remote cache: %s", target.Label, err)
		}
		return nil
	}
	if needsPostBuildFile(target) {
		// Need to explicitly write this guy
		fs.WriteFile(bytes.NewReader(metadata.Stdout), path.Join(target.OutDir(), target.PostBuildOutputFileName()), 0644)
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
