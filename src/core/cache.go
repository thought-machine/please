package core

// Cache is our general interface to caches for built targets.
// The implementations are in //src/cache, but the interface is in this package because
// it's passed around on the BuildState object.
type Cache interface {
	// Stores the results of a single build target.
	Store(target *BuildTarget, key []byte, metadata *BuildMetadata, files []string)
	// Retrieves the results of a single build target.
	// If successful, the outputs will be placed into the output file tree and
	// the returned metadata structure will be populated with whatever is stored
	// (the only field that is guaranteed is standard output though, and only for targets
	// that have post-build functions).
	// If unsuccessful, it will return nil.
	Retrieve(target *BuildTarget, key []byte, files []string) *BuildMetadata
	// Retrieves the results of a test run.
	// Cleans any artifacts associated with this target from the cache, for any possible key.
	// Some implementations may not honour this, depending on configuration etc.
	Clean(target *BuildTarget)
	// Cleans the entire cache.
	CleanAll()
	// Shuts down the cache, blocking until any potentially pending requests are done.
	Shutdown()
}
