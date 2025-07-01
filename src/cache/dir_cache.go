// Directory-based cache.

package cache

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/djherbis/atime"
	"github.com/dustin/go-humanize"

	"github.com/thought-machine/please/src/clean"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

type dirCache struct {
	Dir      string
	Compress bool
	Suffix   string
	mtime    time.Time
	added    map[string]uint64
	mutex    sync.Mutex
}

func (cache *dirCache) Store(target *core.BuildTarget, key []byte, files []string) {
	cacheDir := cache.getPath(target, key, "")
	tmpDir := cache.getFullPath(target, key, "", "=")
	cache.markDir(cacheDir, 0)
	if err := fs.RemoveAll(cacheDir); err != nil {
		log.Warning("Failed to remove existing cache directory %s: %s", cacheDir, err)
		return
	}
	cache.storeFiles(target, key, "", cacheDir, tmpDir, files, true)
	if err := os.Rename(tmpDir, cacheDir); err != nil && !os.IsNotExist(err) {
		log.Warning("Failed to create cache directory %s: %s", cacheDir, err)
	}
}

// storeFiles stores the given files in the cache, either compressed or not.
func (cache *dirCache) storeFiles(target *core.BuildTarget, key []byte, suffix, cacheDir, tmpDir string, files []string, clean bool) {
	var totalSize uint64
	if cache.Compress {
		totalSize = cache.storeCompressed(target, tmpDir, files)
	} else {
		for _, out := range files {
			totalSize += cache.storeFile(target, out, tmpDir)
		}
	}
	cache.markDir(cacheDir, totalSize)
}

// storeCompressed stores all the given files in the cache as a single compressed tarball.
func (cache *dirCache) storeCompressed(target *core.BuildTarget, filename string, files []string) uint64 {
	log.Debug("Storing %s: %s in dir cache...", target.Label, filename)
	if err := cache.storeCompressed2(target, filename, files); err != nil {
		log.Warning("Failed to store files in cache: %s", err)
		fs.RemoveAll(filename) // Just a best-effort removal at this point
		return 0
	}
	// It's too hard to tell from a tar.Writer how big the resulting tarball is. Easier to just re-stat it here.
	info, err := os.Stat(filename)
	if err != nil {
		log.Warning("Can't read stored file: %s", err)
		return 0
	}
	return uint64(info.Size())
}

// storeCompressed2 stores all the given files in the cache as a single compressed tarball.
func (cache *dirCache) storeCompressed2(target *core.BuildTarget, filename string, files []string) error {
	if err := cache.ensureStoreReady(filename); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	defer bw.Flush()
	gw := gzip.NewWriter(bw)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	outDir := target.OutDir()
	for _, file := range files {
		// Any one of these might be a directory, so we have to walk them.
		if err := fs.Walk(filepath.Join(outDir, file), func(name string, isDir bool) error {
			hdr, err := cache.tarHeader(name, outDir)
			if err != nil {
				return err
			} else if err := tw.WriteHeader(hdr); err != nil {
				return err
			} else if hdr.Typeflag != tar.TypeDir && hdr.Typeflag != tar.TypeSymlink {
				f, err := os.Open(name)
				if err != nil {
					return err
				} else if _, err := io.Copy(tw, f); err != nil {
					return err
				}
				f.Close() // Do not defer this, otherwise we can open too many files at once.
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// tarHeader returns an appropriate tar header for the given file.
func (cache *dirCache) tarHeader(file, prefix string) (*tar.Header, error) {
	info, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	link := ""
	if info.Mode()&os.ModeSymlink != 0 {
		// We have to read the link target separately.
		link, err = os.Readlink(file)
		if err != nil {
			return nil, err
		}
	}
	hdr, err := tar.FileInfoHeader(info, link)
	if hdr != nil {
		hdr.Name = strings.TrimLeft(strings.TrimPrefix(file, prefix), "/")
		// Zero out all timestamps.
		hdr.ModTime = cache.mtime
		hdr.AccessTime = cache.mtime
		hdr.ChangeTime = cache.mtime
		// Strip user/group ids.
		hdr.Uid = 0
		hdr.Gid = 0
		// Setting the user/group write bits helps consistency of output.
		hdr.Mode |= 0220
	}
	return hdr, err
}

// ensureStoreReady ensures that the directory containing the given filename exists and any previous file has been removed.
func (cache *dirCache) ensureStoreReady(filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
		return err
	} else if err := fs.RemoveAll(filename); err != nil {
		return err
	}
	return nil
}

func (cache *dirCache) storeFile(target *core.BuildTarget, out, cacheDir string) uint64 {
	log.Debug("Storing %s: %s in dir cache...", target.Label, out)
	outFile := filepath.Join(core.RepoRoot, target.OutDir(), out)
	cachedFile := filepath.Join(cacheDir, out)
	if err := cache.ensureStoreReady(cachedFile); err != nil {
		log.Warning("Failed to setup cache directory: %s", err)
		return 0
	}
	if err := fs.RecursiveLink(outFile, cachedFile); err != nil {
		// Cannot hardlink files into the cache, must copy them for reals.
		log.Warning("Failed to store cache file %s: %s", cachedFile, err)
	}
	// TODO(peterebden): This is a little inefficient, it would be better to track the size in
	//                   RecursiveCopy rather than walking again.
	size, _ := findSize(cachedFile)
	return size
}

func (cache *dirCache) Retrieve(target *core.BuildTarget, key []byte, outs []string) bool {
	return cache.retrieve(target, key, "", outs)
}

// retrieveFiles retrieves the given set of files from the cache.
func (cache *dirCache) retrieve(target *core.BuildTarget, key []byte, suffix string, outs []string) bool {
	found, err := cache.retrieveFiles(target, cache.getPath(target, key, suffix), outs)
	if err != nil && !os.IsNotExist(err) {
		log.Warning("Failed to retrieve %s from dir cache: %s", target.Label, err)
		return false
	} else if found {
		log.Debug("Retrieved %s: %s from dir cache", target.Label, suffix)
	}
	return found
}

func (cache *dirCache) retrieveFiles(target *core.BuildTarget, cacheDir string, outs []string) (bool, error) {
	if !core.PathExists(cacheDir) {
		log.Debug("%s: %s doesn't exist in dir cache", target.Label, cacheDir)
		return false, nil
	}
	cache.markDir(cacheDir, 0)
	if len(outs) == 0 {
		return true, nil
	}
	if cache.Compress {
		log.Debug("Retrieving %s: %s from compressed cache", target.Label, cacheDir)
		return true, cache.retrieveCompressed(target, cacheDir)
	}
	for _, out := range outs {
		realOut, err := cache.ensureRetrieveReady(target, out)
		if err != nil {
			return false, err
		}
		cachedOut := filepath.Join(cacheDir, out)
		log.Debug("Retrieving %s: %s from dir cache...", target.Label, cachedOut)
		if err := fs.RecursiveLink(cachedOut, realOut); err != nil {
			return false, err
		}
	}
	return true, nil
}

// retrieveCompressed retrieves the given outs from a compressed tarball.
// Right now it retrieves everything from the file which is sort of slightly incorrect but in practice
// we should get away with it (because changing the set of outputs from what was stored would also change
// the hash, so theoretically at least the two should line up).
func (cache *dirCache) retrieveCompressed(target *core.BuildTarget, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break // End of archive
			}
			return err
		}
		out, err := cache.ensureRetrieveReady(target, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			// Just create the directory
			if err := os.MkdirAll(out, core.DirPermissions); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, out); err != nil {
				return err
			}
		default:
			f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			// N.B. It is important not to defer this - since defers do not run until the function
			//      exits, we can stack up many open files within this loop, and when retrieving multiple
			//      large artifacts at once can easily run out of file handles.
			f.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureRetrieveReady makes sure that appropriate directories are created and old outputs are removed.
func (cache *dirCache) ensureRetrieveReady(target *core.BuildTarget, out string) (string, error) {
	fullOut := filepath.Join(core.RepoRoot, target.OutDir(), out)
	if strings.ContainsRune(out, '/') { // The root directory will be there, only need to worry about outs in subdirectories.
		if err := os.MkdirAll(filepath.Dir(fullOut), core.DirPermissions); err != nil {
			return "", err
		}
	}
	// It seems to be quite important that we unlink the existing file first to avoid ETXTBSY errors
	// in cases where we're running an existing binary (as Please does during bootstrap, for example).
	if err := fs.RemoveAll(fullOut); err != nil {
		return "", err
	}
	return fullOut, nil
}

func (cache *dirCache) Clean(target *core.BuildTarget) {
	// Remove for all possible keys, so can't get getPath here
	if err := fs.RemoveAll(filepath.Join(cache.Dir, target.Label.PackageName, target.Label.Name)); err != nil {
		log.Warning("Failed to remove artifacts for %s from dir cache: %s", target.Label, err)
	}
}

func (cache *dirCache) CleanAll() {
	if err := clean.AsyncDeleteDir(cache.Dir); err != nil {
		log.Error("Failed to clean cache: %s", err)
	}
}

func (cache *dirCache) Shutdown() {}

func (cache *dirCache) getPath(target *core.BuildTarget, key []byte, extra string) string {
	return cache.getFullPath(target, key, extra, "")
}

func (cache *dirCache) getFullPath(target *core.BuildTarget, key []byte, extra, suffix string) string {
	// The extra identifier is not needed for non-compressed caches.
	if !cache.Compress {
		extra = ""
	} else {
		extra = strings.ReplaceAll(extra, "/", "_")
	}
	// NB. Is very important to use a padded encoding here so lengths are consistent when cleaning.
	return filepath.Join(cache.Dir, target.Label.PackageName, target.Label.Name, base64.URLEncoding.EncodeToString(key)) + extra + suffix + cache.Suffix
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
		Compress: config.Cache.DirCompress,
		Dir:      config.Cache.Dir,
		added:    map[string]uint64{},
		mtime:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
	if cache.Compress {
		cache.Suffix = ".tar.gz"
	}
	// Absolute paths are allowed. Relative paths are interpreted relative to the repo root.
	if !filepath.IsAbs(config.Cache.Dir) {
		cache.Dir = filepath.Join(core.RepoRoot, config.Cache.Dir)
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
		if cache.shouldClean(name, isDir) {
			if size, marked := cache.isMarked(path); marked {
				totalSize += size
				if !cache.Compress {
					return filepath.SkipDir // Already handled
				}
				return nil // Need to keep walking if we are dealing with compressed files
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
			if !cache.Compress {
				return filepath.SkipDir
			}
		}
		return nil // nothing particularly to do for other entries
	}); err != nil {
		log.Error("error walking cache directory: %s\n", err)
		return totalSize
	}
	log.Info("Total cache size: %s", humanize.Bytes(totalSize))
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

		log.Debug("Cleaning %s, accessed %s, saves %s", entry.Path, humanize.Time(time.Unix(entry.Atime, 0)), humanize.Bytes(entry.Size))
		// Try to rename the directory first so we don't delete bits while someone might access them.
		newPath := entry.Path + "="
		if err := os.Rename(entry.Path, newPath); err != nil {
			log.Errorf("Couldn't rename %s: %s", entry.Path, err)
			continue
		}
		if err := fs.RemoveAll(newPath); err != nil {
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

// shouldClean returns true if we should clean this file.
// We track this in order to clean only entire entries in the cache, not just individual files from them.
func (cache *dirCache) shouldClean(name string, isDir bool) bool {
	if cache.Compress == isDir {
		return false // If we're compressing, don't look for directories. If we're not, only look at directories.
	} else if !strings.HasSuffix(name, cache.Suffix) {
		return false // Suffix must match.
	}
	name = strings.TrimSuffix(name, cache.Suffix)
	// 28 == length of 20-byte sha1 hash, encoded to base64, which always gets a trailing =
	// as padding so we can check that to be "sure".
	// Also 29 in case we appended an extra = (which we do for temporary files that are still being written to)
	// Similarly for sha256 which is length 44.
	return ((len(name) == 28 || len(name) == 29) && name[27] == '=') || ((len(name) == 44 || len(name) == 45) && name[43] == '=')
}
