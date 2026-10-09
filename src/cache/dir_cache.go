// Directory-based cache.

package cache

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

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
	lockFile, err := core.AcquireExclusiveFileLock(cacheDir + ".lock")
	if err != nil {
		log.Warning("Failed to acquire cache lock for %s, will not store", target)
		return
	}
	defer core.ReleaseFileLock(lockFile)

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
	return unsharedSize(cachedFile)
}

func (cache *dirCache) Retrieve(target *core.BuildTarget, key []byte, outs []string) bool {
	lockFile, err := core.AcquireSharedFileLock(cache.getPath(target, key, "") + ".lock")
	if err != nil {
		log.Warning("Failed to acquire cache lock for %s, will not retrieve", target)
		return false
	}
	defer core.ReleaseFileLock(lockFile)

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
	info, err := os.Stat(cacheDir)
	if os.IsNotExist(err) {
		log.Debug("%s: %s doesn't exist in dir cache", target.Label, cacheDir)
		return false, nil
	} else if err != nil {
		return false, err
	}
	cache.markDir(cacheDir, 0)
	cache.touch(cacheDir, info)
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

// Period of time in seconds between which two artifacts are considered to have been used at the same time.
const accessTimeGracePeriod = 600 // Ten minutes

// touchInterval is the minimum time between updates of an entry's modification time when it's retrieved.
// Updating it on every retrieval would cause a lot of unnecessary writes.
const touchInterval = time.Hour

// A cacheEntry represents a single entry in the cache (i.e. one stored target).
type cacheEntry struct {
	Path  string
	Size  uint64 // Total size of the files in this entry that are not linked from outside the cache
	Mtime int64
	Files []fileID
}

// A fileID identifies a file on disk independently of the path(s) it's linked at.
type fileID struct {
	Dev, Ino uint64
}

// A cacheFile is a single file in the cache, which may be linked into multiple entries
// and also from outside the cache (typically into plz-out, when it's stored or retrieved).
type cacheFile struct {
	Size     uint64
	Links    uint64 // Total number of hard links to this file
	Refs     uint64 // Number of those links that are inside the cache
	Unshared bool   // True if this file is only linked from inside the cache.
}

// fileInfo returns the identifier and number of hard links for a file.
func fileInfo(info os.FileInfo) (fileID, uint64) {
	st := info.Sys().(*syscall.Stat_t)
	// The types of these fields vary between platforms, so the conversions are needed on some of them.
	return fileID{Dev: uint64(st.Dev), Ino: uint64(st.Ino)}, uint64(st.Nlink) //nolint:unconvert
}

// unsharedSize returns the total size of the files under the given path that are not hard linked from anywhere else.
func unsharedSize(path string) uint64 {
	var size uint64
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if _, links := fileInfo(info); links <= 1 {
				size += uint64(info.Size())
			}
		}
		return nil
	})
	return size
}

// readEntry reads a single entry in the cache, recording all the files in it.
func readEntry(path string, files map[fileID]*cacheFile) (*cacheEntry, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	entry := &cacheEntry{Path: path, Mtime: info.ModTime().Unix()}
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if info.IsDir() {
			return nil
		}
		id, links := fileInfo(info)
		if _, present := files[id]; !present {
			files[id] = &cacheFile{Size: uint64(info.Size()), Links: links}
		}
		files[id].Refs++
		entry.Files = append(entry.Files, id)
		return nil
	}); err != nil {
		return nil, err
	}
	return entry, nil
}

// clean runs background cleaning of this cache until the process exits.
// Returns the total size of the cache after it's finished.
// Sizes only count files that are not hard linked from outside the cache; those are typically also in someone's
// plz-out, and so removing them would not free any space (and would make it less likely for that file to be shared
// with the next repo that builds it).
func (cache *dirCache) clean(highWaterMark, lowWaterMark uint64) uint64 {
	entries := []*cacheEntry{}
	files := map[fileID]*cacheFile{}
	var totalSize uint64
	if err := fs.Walk(cache.Dir, func(path string, isDir bool) error {
		if !cache.shouldClean(filepath.Base(path), isDir) {
			return nil // nothing particularly to do for other entries
		}
		if size, marked := cache.isMarked(path); marked {
			totalSize += size
		} else if entry, err := readEntry(path, files); err != nil {
			// This can happen if another process is cleaning concurrently; it's not fatal to us.
			log.Warning("Failed to read cache entry %s: %s", path, err)
		} else {
			entries = append(entries, entry)
		}
		if !cache.Compress {
			return filepath.SkipDir // Already handled
		}
		return nil // Need to keep walking if we are dealing with compressed files
	}); err != nil {
		log.Error("error walking cache directory: %s\n", err)
		return totalSize
	}
	for _, f := range files {
		if f.Unshared = f.Links <= f.Refs; f.Unshared {
			totalSize += f.Size
		}
	}
	// Entries with no unshared files would free nothing if removed, so are never candidates for cleaning.
	candidates := entries[:0]
	for _, entry := range entries {
		for _, id := range entry.Files {
			if f := files[id]; f.Unshared {
				entry.Size += f.Size
			}
		}
		if entry.Size > 0 {
			candidates = append(candidates, entry)
		}
	}
	log.Info("Total cache size: %s", humanize.Bytes(totalSize))
	if totalSize < highWaterMark {
		return totalSize // Nothing to do, cache is small enough.
	}
	// OK, we need to slim it down a bit. We implement a simple LRU algorithm.
	sort.Slice(candidates, func(i, j int) bool {
		diff := candidates[i].Mtime - candidates[j].Mtime
		if diff > -accessTimeGracePeriod && diff < accessTimeGracePeriod {
			return candidates[i].Size > candidates[j].Size
		}
		return candidates[i].Mtime < candidates[j].Mtime
	})
	for _, entry := range candidates {
		if _, marked := cache.isMarked(entry.Path); marked {
			continue
		}

		log.Debug("Cleaning %s, used %s, saves up to %s", entry.Path, humanize.Time(time.Unix(entry.Mtime, 0)), humanize.Bytes(entry.Size))
		if err := cache.cleanPath(entry.Path); err != nil {
			log.Warning("Error while cleaning cache: %s", err)
			continue
		}
		// A file is only freed once the last entry referring to it is removed.
		for _, id := range entry.Files {
			f := files[id]
			f.Refs--
			if f.Refs == 0 && f.Unshared {
				totalSize -= f.Size
			}
		}
		if totalSize < lowWaterMark {
			break
		}
	}
	return totalSize
}

// touch updates the modification time of a cache entry to mark it as recently used.
func (cache *dirCache) touch(path string, info os.FileInfo) {
	if time.Since(info.ModTime()) < touchInterval {
		return // Updated recently enough already
	}
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		log.Debug("Failed to update modification time of %s: %s", path, err)
	}
}

func (cache *dirCache) cleanPath(path string) error {
	lockFile, err := core.AcquireExclusiveFileLock(path + ".lock")
	if err != nil {
		return err
	}
	defer core.ReleaseFileLock(lockFile)

	// Try to rename the directory first so if anything goes wrong we leave it inaccessible to anyone else
	newPath := path + "="
	if err := os.Rename(path, newPath); err != nil {
		return fmt.Errorf("Couldn't rename %s: %w", path, err)
	}
	if err := fs.RemoveAll(newPath); err != nil {
		return fmt.Errorf("Couldn't remove %s: %w", newPath, err)
	}
	return nil
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
