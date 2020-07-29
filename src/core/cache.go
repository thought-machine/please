package core

import (
	"io"
)

// Cache is our general interface to caches for built targets.
// The implementations are in //src/cache, but the interface is in this package because
// it's passed around on the BuildState object.
type Cache interface {
	// Stores the results of a single build target.
	Store(target *BuildTarget, key []byte, files []string)
	// Retrieves the results of a single build target.
	// If successful, the outputs will be placed into the output file tree and
	// the returned metadata structure will be populated with whatever is stored
	// (the only field that is guaranteed is standard output though, and only for targets
	// that have post-build functions).
	// If unsuccessful, it will return nil.
	Retrieve(target *BuildTarget, key []byte, files []string) bool
	// Stores the contents of a single file from the given reader.
	StoreFile(target *BuildTarget, key []byte, contents io.Reader, filename string)
	// Retrieves the contents of a single file for a build target.
	// The returned reader is nil if it couldn't be retrieved.
	RetrieveFile(target *BuildTarget, key []byte, filename string) io.ReadCloser
	// Cleans any artifacts associated with this target from the cache, for any possible key.
	// Some implementations may not honour this, depending on configuration etc.
	Clean(target *BuildTarget)
	// Cleans the entire cache.
	CleanAll()
	// Shuts down the cache, blocking until any potentially pending requests are done.
	Shutdown()
}
