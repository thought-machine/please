package core

// Cache is our general interface to caches for built targets.
// The implementations are in //src/cache, but the interface is in this package because
// it's passed around on the BuildState object.
type Cache interface {
	// Stores the results of a single build target.
	// Optionally can store extra files against it at the same time.
	Store(target *BuildTarget, key []byte, files ...string)
	// Stores an extra file against a build target.
	// The file name is relative to the target's out directory.
	StoreExtra(target *BuildTarget, key []byte, file string)
	// Retrieves the results of a single build target.
	// If successful, the outputs will be placed into the output file tree.
	Retrieve(target *BuildTarget, key []byte) bool
	// Retrieves an extra file previously stored by StoreExtra.
	// If successful, the file will be placed into the output file tree.
	RetrieveExtra(target *BuildTarget, key []byte, file string) bool
	// Cleans any artifacts associated with this target from the cache, for any possible key.
	Clean(target *BuildTarget)
	// Cleans the entire cache.
	CleanAll()
	// Shuts down the cache, blocking until any potentially pending requests are done.
	Shutdown()
}
