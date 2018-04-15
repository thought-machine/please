// Diretory-based cache.

package cache

import (
	"encoding/base64"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/djherbis/atime"
	"github.com/dustin/go-humanize"

	"core"
	"fs"
)

type dirCache struct {
	Dir   string
	added map[string]uint64
	mutex sync.Mutex
}

func (cache *dirCache) Store(target *core.BuildTarget, key []byte, files ...string) {
	cacheDir := cache.getPath(target, key)
	tmpDir := cacheDir + "=" // Temp dir which we'll move when it's ready.
	cache.markDir(cacheDir, 0)
	// Clear out anything that might already be there.
	if err := os.RemoveAll(cacheDir); err != nil {
		log.Warning("Failed to remove existing cache directory %s: %s", cacheDir, err)
		return
	} else if err := os.MkdirAll(tmpDir, core.DirPermissions); err != nil {
		log.Warning("Failed to create cache directory %s: %s", tmpDir, err)
		return
	}
	var totalSize uint64
	for out := range cacheArtifacts(target, files...) {
		totalSize += cache.storeFile(target, out, tmpDir)
	}
	cache.markDir(cacheDir, totalSize)
	if err := os.Rename(tmpDir, cacheDir); err != nil {
		log.Warning("Failed to create cache directory %s: %s", cacheDir, err)
	}
}

func (cache *dirCache) StoreExtra(target *core.BuildTarget, key []byte, out string) {
	path := cache.getPath(target, key)
	cache.markDir(path, 0)
	size := cache.storeFile(target, out, path)
	cache.markDir(path, size)
}

func (cache *dirCache) storeFile(target *core.BuildTarget, out, cacheDir string) uint64 {
	log.Debug("Storing %s: %s in dir cache...", target.Label, out)
	if dir := path.Dir(out); dir != "." {
		if err := os.MkdirAll(path.Join(cacheDir, dir), core.DirPermissions); err != nil {
			log.Warning("Failed to create cache directory %s: %s", path.Join(cacheDir, dir), err)
			return 0
		}
	}
	outFile := path.Join(core.RepoRoot, target.OutDir(), out)
	cachedFile := path.Join(cacheDir, out)
	// Remove anything existing
	if err := os.RemoveAll(cachedFile); err != nil {
		log.Warning("Failed to remove existing cached file %s: %s", cachedFile, err)
	} else if err := os.MkdirAll(cacheDir, core.DirPermissions); err != nil {
		log.Warning("Failed to create cache directory %s: %s", cacheDir, err)
		return 0
	} else if err := core.RecursiveCopyFile(outFile, cachedFile, fileMode(target), true, true); err != nil {
		// Cannot hardlink files into the cache, must copy them for reals.
		log.Warning("Failed to store cache file %s: %s", cachedFile, err)
	}
	// TODO(peterebden): This is a little inefficient, it would be better to track the size in
	//                   RecursiveCopyFile rather than walking again.
	size, _ := findSize(cachedFile)
	return size
}

func (cache *dirCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	cacheDir := cache.getPath(target, key)
	if !core.PathExists(cacheDir) {
		log.Debug("%s: %s doesn't exist in dir cache", target.Label, cacheDir)
		return false
	}
	cache.markDir(cacheDir, 0)
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
	cache.markDir(cacheDir, 0)
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

// markDir marks a directory as added to the cache, which saves it from later deletion.
func (cache *dirCache) markDir(path string, size uint64) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.added[path] = size
	cache.added[path+"="] = size
}

// isMarked returns true if a directory has previously been passed to markDir.
func (cache *dirCache) isMarked(path string) (uint64, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	size, present := cache.added[path]
	return size, present
}

func newDirCache(config *core.Configuration) *dirCache {
	cache := &dirCache{
		added: map[string]uint64{},
	}
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
	// Start the cache-cleaning goroutine.
	if config.Cache.DirClean {
		go cache.clean(uint64(config.Cache.DirCacheHighWaterMark), uint64(config.Cache.DirCacheLowWaterMark))
	}
	return cache
}

func fileMode(target *core.BuildTarget) os.FileMode {
	if target.IsBinary {
		return 0555
	}
	return 0444
}

// Period of time in seconds between which two artifacts are considered to have the same atime.
const accessTimeGracePeriod = 600 // Ten minutes

// A cacheEntry represents a single file entry in the cache.
type cacheEntry struct {
	Path  string
	Size  uint64
	Atime int64
}

func findSize(path string) (uint64, error) {
	var totalSize uint64
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		totalSize += uint64(info.Size())
		return nil
	}); err != nil {
		return 0, err
	}
	return totalSize, nil
}

// clean runs background cleaning of this cache until the process exits.
// Returns the total size of the cache after it's finished.
func (cache *dirCache) clean(highWaterMark, lowWaterMark uint64) uint64 {
	entries := []cacheEntry{}
	var totalSize uint64
	if err := fs.Walk(cache.Dir, func(path string, isDir bool) error {
		name := filepath.Base(path)
		if isDir && (len(name) == 28 || len(name) == 29) && name[27] == '=' {
			// Directory has the right length. We do this in an attempt to clean only entire
			// entries in the cache, not just individual files from them.
			// 28 == length of 20-byte sha1 hash, encoded to base64, which always gets a trailing =
			// as padding so we can check that to be "sure".
			// Also 29 in case we appended an extra = (see below)
			if size, marked := cache.isMarked(path); marked {
				totalSize += size
				return filepath.SkipDir // Already handled
			}
			size, err := findSize(path)
			if err != nil {
				return err
			}
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			entries = append(entries, cacheEntry{
				Path:  path,
				Size:  size,
				Atime: atime.Get(info).Unix(),
			})
			totalSize += size
			return filepath.SkipDir
		}
		return nil // nothing particularly to do for other entries
	}); err != nil {
		log.Error("error walking cache directory: %s\n", err)
		return totalSize
	}
	log.Info("Total cache size: %s", humanize.Bytes(uint64(totalSize)))
	if totalSize < highWaterMark {
		return totalSize // Nothing to do, cache is small enough.
	}
	// OK, we need to slim it down a bit. We implement a simple LRU algorithm.
	sort.Slice(entries, func(i, j int) bool {
		diff := entries[i].Atime - entries[j].Atime
		if diff > -accessTimeGracePeriod && diff < accessTimeGracePeriod {
			return entries[i].Size > entries[j].Size
		}
		return entries[i].Atime < entries[j].Atime
	})
	for _, entry := range entries {
		if _, marked := cache.isMarked(entry.Path); marked {
			continue
		}

		log.Debug("Cleaning %s, accessed %s, saves %s", entry.Path, humanize.Time(time.Unix(entry.Atime, 0)), humanize.Bytes(uint64(entry.Size)))
		// Try to rename the directory first so we don't delete bits while someone might access them.
		newPath := entry.Path + "="
		if err := os.Rename(entry.Path, newPath); err != nil {
			log.Errorf("Couldn't rename %s: %s", entry.Path, err)
			continue
		}
		if err := os.RemoveAll(newPath); err != nil {
			log.Errorf("Couldn't remove %s: %s", newPath, err)
			continue
		}
		totalSize -= entry.Size
		if totalSize < lowWaterMark {
			break
		}
	}
	return totalSize
}
