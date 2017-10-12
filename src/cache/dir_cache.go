// Diretory-based cache.

package cache

import (
	"encoding/base64"
	"os"
	"path"
	"syscall"

	"core"
)

type dirCache struct {
	Dir string
}

func (cache *dirCache) Store(target *core.BuildTarget, key []byte, files ...string) {
	cacheDir := cache.getPath(target, key)
	tmpDir := cacheDir + "=" // Temp dir which we'll move when it's ready.
	// Clear out anything that might already be there.
	if err := os.RemoveAll(cacheDir); err != nil {
		log.Warning("Failed to remove existing cache directory %s: %s", cacheDir, err)
		return
	} else if err := os.MkdirAll(tmpDir, core.DirPermissions); err != nil {
		log.Warning("Failed to create cache directory %s: %s", tmpDir, err)
		return
	}
	for out := range cacheArtifacts(target, files...) {
		cache.storeFile(target, out, tmpDir)
	}
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		log.Warning("Failed to create cache directory %s: %s", cacheDir, err)
	}
}

func (cache *dirCache) StoreExtra(target *core.BuildTarget, key []byte, out string) {
	cache.storeFile(target, out, cache.getPath(target, key))
}

func (cache *dirCache) storeFile(target *core.BuildTarget, out, cacheDir string) {
	log.Debug("Storing %s: %s in dir cache...", target.Label, out)
	if dir := path.Dir(out); dir != "." {
		if err := os.MkdirAll(path.Join(cacheDir, dir), core.DirPermissions); err != nil {
			log.Warning("Failed to create cache directory %s: %s", path.Join(cacheDir, dir), err)
			return
		}
	}
	outFile := path.Join(core.RepoRoot, target.OutDir(), out)
	cachedFile := path.Join(cacheDir, out)
	// Remove anything existing
	if err := os.RemoveAll(cachedFile); err != nil {
		log.Warning("Failed to remove existing cached file %s: %s", cachedFile, err)
	} else if err := os.MkdirAll(cacheDir, core.DirPermissions); err != nil {
		log.Warning("Failed to create cache directory %s: %s", cacheDir, err)
		return
	} else if err := core.RecursiveCopyFile(outFile, cachedFile, fileMode(target), true, true); err != nil {
		// Cannot hardlink files into the cache, must copy them for reals.
		log.Warning("Failed to store cache file %s: %s", cachedFile, err)
	}
}

func (cache *dirCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	cacheDir := cache.getPath(target, key)
	if !core.PathExists(cacheDir) {
		log.Debug("%s: %s doesn't exist in dir cache", target.Label, cacheDir)
		return false
	}
	for out := range cacheArtifacts(target) {
		if !cache.RetrieveExtra(target, key, out) {
			return false
		}
	}
	return true
}

func (cache *dirCache) RetrieveExtra(target *core.BuildTarget, key []byte, out string) bool {
	outDir := path.Join(core.RepoRoot, target.OutDir())
	cacheDir := cache.getPath(target, key)
	cachedOut := path.Join(cacheDir, out)
	realOut := path.Join(outDir, out)
	if !core.PathExists(cachedOut) {
		log.Debug("%s: %s doesn't exist in dir cache", target.Label, cachedOut)
		return false
	}
	log.Debug("Retrieving %s: %s from dir cache...", target.Label, cachedOut)
	if dir := path.Dir(realOut); dir != "." {
		if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
			log.Warning("Failed to create output directory %s: %s", dir, err)
			return false
		}
	}
	// It seems to be quite important that we unlink the existing file first to avoid ETXTBSY errors
	// in cases where we're running an existing binary (as Please does during bootstrap, for example).
	if err := os.RemoveAll(realOut); err != nil {
		log.Warning("Failed to unlink existing output %s: %s", realOut, err)
		return false
	}
	// Recursively hardlink files back out of the cache
	if err := core.RecursiveCopyFile(cachedOut, realOut, fileMode(target), true, true); err != nil {
		log.Warning("Failed to move cached file to output: %s -> %s: %s", cachedOut, realOut, err)
		return false
	}
	log.Debug("Retrieved %s: %s from dir cache", target.Label, cachedOut)
	return true
}

func (cache *dirCache) Clean(target *core.BuildTarget) {
	// Remove for all possible keys, so can't get getPath here
	if err := os.RemoveAll(path.Join(cache.Dir, target.Label.PackageName, target.Label.Name)); err != nil {
		log.Warning("Failed to remove artifacts for %s from dir cache: %s", target.Label, err)
	}
}

func (cache *dirCache) CleanAll() {
	if err := core.AsyncDeleteDir(cache.Dir); err != nil {
		log.Error("Failed to clean cache: %s", err)
	}
}

func (cache *dirCache) Shutdown() {}

func (cache *dirCache) getPath(target *core.BuildTarget, key []byte) string {
	// NB. Is very important to use a padded encoding here so lengths are consistent for cache_cleaner.
	return path.Join(cache.Dir, target.Label.PackageName, target.Label.Name, base64.URLEncoding.EncodeToString(key))
}

func newDirCache(config *core.Configuration) *dirCache {
	cache := new(dirCache)
	// Absolute paths are allowed. Relative paths are interpreted relative to the repo root.
	if config.Cache.Dir[0] == '/' {
		cache.Dir = config.Cache.Dir
	} else {
		cache.Dir = path.Join(core.RepoRoot, config.Cache.Dir)
	}
	// Make directory if it doesn't exist.
	if err := os.MkdirAll(cache.Dir, core.DirPermissions); err != nil {
		log.Fatalf("Failed to create root cache directory %s: %s", cache.Dir, err)
	}
	// Fire off the cache cleaner process.
	if config.Cache.DirCacheCleaner != "" && config.Cache.DirCacheCleaner != "none" {
		go func() {
			cleaner := core.ExpandHomePath(config.Cache.DirCacheCleaner)
			log.Info("Running cache cleaner: %s --dir %s --high_water_mark %s --low_water_mark %s",
				cleaner, cache.Dir, config.Cache.DirCacheHighWaterMark, config.Cache.DirCacheLowWaterMark)
			if _, err := syscall.ForkExec(cleaner, []string{
				cleaner,
				"--dir", cache.Dir,
				"--high_water_mark", config.Cache.DirCacheHighWaterMark,
				"--low_water_mark", config.Cache.DirCacheLowWaterMark,
			}, nil); err != nil {
				log.Errorf("Failed to start cache cleaner: %s", err)
			}
		}()
	}
	return cache
}

func fileMode(target *core.BuildTarget) os.FileMode {
	if target.IsBinary {
		return 0555
	}
	return 0444
}
