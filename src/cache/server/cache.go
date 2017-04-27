// Package cache_server contains core functionality for our cache servers; storing & retrieving files etc.
package server

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/djherbis/atime"
	"github.com/dustin/go-humanize"
	"github.com/streamrail/concurrent-map"

	"core"
)

// A cachedFile stores metadata about a file stored in our cache.
type cachedFile struct {
	// Arbitrates single access to this file
	sync.RWMutex
	// Time the file was last read
	lastReadTime time.Time
	// Number of times the file has been read
	readCount int
	// Size of the file
	size int64
}

// A Cache is the underlying implementation of our HTTP and RPC caches that handles storing & retrieving artifacts.
type Cache struct {
	cachedFiles cmap.ConcurrentMap
	totalSize   int64
	rootPath    string
}

// NewCache initialises the cache and fires off a background cleaner goroutine which runs every
// cleanFrequency seconds. The high and low water marks control a (soft) max size and a (harder)
// minimum size.
func NewCache(path string, cleanFrequency, maxArtifactAge time.Duration, lowWaterMark, highWaterMark uint64) *Cache {
	log.Notice("Initialising cache with settings:\n  Path: %s\n  Clean frequency: %s\n  Max artifact age: %s\n  Low water mark: %s\n  High water mark: %s",
		path, cleanFrequency, maxArtifactAge, humanize.Bytes(lowWaterMark), humanize.Bytes(highWaterMark))
	cache := newCache(path)
	go cache.clean(cleanFrequency, maxArtifactAge, int64(lowWaterMark), int64(highWaterMark))
	return cache
}

// newCache is an internal constructor intended mostly for testing. It doesn't start the cleaner goroutine.
func newCache(path string) *Cache {
	cache := &Cache{rootPath: path}
	cache.scan()
	return cache
}

// TotalSize returns the current total size monitored by the cache, in bytes.
func (cache *Cache) TotalSize() int64 {
	return cache.totalSize
}

// NumFiles returns the number of files currently monitored by the cache.
func (cache *Cache) NumFiles() int {
	return cache.cachedFiles.Count()
}

// scan scans the directory tree for files.
func (cache *Cache) scan() {
	cache.cachedFiles = cmap.New()
	cache.totalSize = 0

	if !core.PathExists(cache.rootPath) {
		if err := os.MkdirAll(cache.rootPath, core.DirPermissions); err != nil {
			log.Fatalf("Failed to create cache directory %s: %s", cache.rootPath, err)
		}
		return
	}

	log.Info("Scanning cache directory %s...", cache.rootPath)
	filepath.Walk(cache.rootPath, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("%s", err)
		} else if !info.IsDir() { // We don't have directory entries.
			name = name[len(cache.rootPath)+1:]
			log.Debug("Found file %s", name)
			size := info.Size()
			cache.cachedFiles.Set(name, &cachedFile{
				lastReadTime: atime.Get(info),
				readCount:    0,
				size:         size,
			})
			cache.totalSize += size
		}
		return nil
	})
	log.Info("Scan complete, found %d entries", cache.cachedFiles.Count())
}

// lockFile locks a file for reading or writing.
// It returns a locked mutex corresponding to that file or nil if there is none.
// The caller should .Unlock() the mutex once they're done with it.
func (cache *Cache) lockFile(path string, write bool, size int64) *cachedFile {
	filei, present := cache.cachedFiles.Get(path)
	var file *cachedFile
	if !present {
		// If we're writing we insert a new one, if we're reading we don't.
		if !write {
			return nil
		}
		file = &cachedFile{
			readCount: 0,
			size:      size,
		}
		file.Lock()
		cache.cachedFiles.Set(path, file)
		atomic.AddInt64(&cache.totalSize, size)
	} else {
		file = filei.(*cachedFile)
		if write {
			file.Lock()
		} else {
			file.RLock()
			file.readCount++
		}
	}
	file.lastReadTime = time.Now()
	return file
}

// removeFile deletes a file from the cache map. It does not remove the on-disk file.
func (cache *Cache) removeFile(path string, file *cachedFile) {
	cache.cachedFiles.Remove(path)
	atomic.AddInt64(&cache.totalSize, -file.size)
	log.Debug("Removing file %s, saves %d, new size will be %d", path, file.size, cache.totalSize)
}

// removeAndDeleteFile deletes a file from the cache map and on-disk.
func (cache *Cache) removeAndDeleteFile(p string, file *cachedFile) {
	cache.removeFile(p, file)
	p = path.Join(cache.rootPath, p)
	if err := os.RemoveAll(p); err != nil {
		log.Error("Failed to delete file: %s", p)
	}
}

// RetrieveArtifact takes in the artifact path as a parameter and checks in the base server
// file directory to see if the file exists in the given path. If found, the function will
// return whatever's been stored there, which might be a directory and therefore contain
// multiple files to be returned.
func (cache *Cache) RetrieveArtifact(artPath string) (map[string][]byte, error) {
	ret := map[string][]byte{}
	if core.IsGlob(artPath) {
		for _, art := range core.Glob(cache.rootPath, []string{artPath}, nil, nil, true) {
			fullPath := path.Join(cache.rootPath, art)
			lock := cache.lockFile(fullPath, false, 0)
			body, err := ioutil.ReadFile(fullPath)
			if lock != nil {
				lock.RUnlock()
			}
			if err != nil {
				return nil, err
			}
			ret[art] = body
		}
		return ret, nil
	}

	fullPath := path.Join(cache.rootPath, artPath)
	lock := cache.lockFile(artPath, false, 0)
	if lock == nil {
		// Can happen if artPath is a directory; we only store artifacts as files.
		// (This is a debatable choice; it's a bit crap either way).
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			return cache.retrieveDir(artPath)
		}
		return nil, os.ErrNotExist
	}
	defer lock.RUnlock()

	if err := filepath.Walk(fullPath, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !info.IsDir() {
			body, err := ioutil.ReadFile(name)
			if err != nil {
				return err
			}
			ret[name[len(cache.rootPath)+1:]] = body
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

// retrieveDir retrieves a directory of artifacts. We don't track the directory itself
// but allow its traversal to retrieve them.
func (cache *Cache) retrieveDir(artPath string) (map[string][]byte, error) {
	log.Debug("Searching dir %s for artifacts", artPath)
	ret := map[string][]byte{}
	fullPath := path.Join(cache.rootPath, artPath)
	err := filepath.Walk(fullPath, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !info.IsDir() {
			// Must strip cache path off the front of this.
			m, err := cache.RetrieveArtifact(name[len(cache.rootPath)+1:])
			if err != nil {
				return err
			}
			for k, v := range m {
				ret[k] = v
			}
		}
		return nil
	})
	return ret, err
}

// StoreArtifact takes in the artifact content and path as parameters and creates a file with
// the given content in the given path.
// The function will return the first error found in the process, or nil if the process is successful.
func (cache *Cache) StoreArtifact(artPath string, key []byte) error {
	log.Info("Storing artifact %s", artPath)
	lock := cache.lockFile(artPath, true, int64(len(key)))
	defer lock.Unlock()

	fullPath := path.Join(cache.rootPath, artPath)
	dirPath := path.Dir(fullPath)
	if err := os.MkdirAll(dirPath, core.DirPermissions); err != nil {
		log.Warning("Couldn't create path %s in http cache: %s", dirPath, err)
		cache.removeAndDeleteFile(artPath, lock)
		os.RemoveAll(dirPath)
		return err
	}
	log.Debug("Writing artifact to %s", fullPath)
	if err := core.WriteFile(bytes.NewReader(key), fullPath, 0); err != nil {
		log.Errorf("Could not create %s artifact: %s", fullPath, err)
		cache.removeAndDeleteFile(artPath, lock)
		return err
	}
	return nil
}

// DeleteArtifact takes in the artifact path as a parameter and removes the artifact from disk.
// The function will return the first error found in the process, or nil if the process is successful.
func (cache *Cache) DeleteArtifact(artPath string) error {
	log.Info("Deleting artifact %s", artPath)
	// We need to search the entire map for prefixes. Pessimism follows...
	paths := cachedFilePaths{}
	for t := range cache.cachedFiles.IterBuffered() {
		if strings.HasPrefix(t.Key, artPath) {
			paths = append(paths, cachedFilePath{file: t.Val.(*cachedFile), path: t.Key})
		}
	}
	// NB. We can't do this in the loop above because there's a risk of deadlock.
	//     We create the temporary slice in preference to calling .Items() and duplicating
	//     the entire map.
	for _, p := range paths {
		p.file.Lock()
		cache.removeFile(p.path, p.file)
		p.file.Unlock()
	}
	return os.RemoveAll(path.Join(cache.rootPath, artPath))
}

// DeleteAllArtifacts will remove all files in the cache directory.
// The function will return the first error found in the process, or nil if the process is successful.
func (cache *Cache) DeleteAllArtifacts() error {
	// Empty entire cache now.
	cache.cachedFiles = cmap.New()
	cache.totalSize = 0
	// Move directory somewhere else
	tempPath := cache.rootPath + "_deleting"
	if err := os.Rename(cache.rootPath, tempPath); err != nil {
		return err
	}
	return os.RemoveAll(tempPath)
}

// clean implements a periodic clean of the cache to remove old artifacts.
func (cache *Cache) clean(cleanFrequency, maxArtifactAge time.Duration, lowWaterMark, highWaterMark int64) {
	for range time.NewTicker(cleanFrequency).C {
		cache.cleanOldFiles(maxArtifactAge)
		cache.singleClean(lowWaterMark, highWaterMark)
	}
}

// cleanOldFiles cleans any files whose last access time is older than the given duration.
func (cache *Cache) cleanOldFiles(maxArtifactAge time.Duration) bool {
	log.Debug("Searching for old files...")
	oldestTime := time.Now().Add(-maxArtifactAge)
	cleaned := 0
	for t := range cache.cachedFiles.IterBuffered() {
		f := t.Val.(*cachedFile)
		if f.lastReadTime.Before(oldestTime) {
			lock := cache.lockFile(t.Key, true, f.size)
			cache.removeAndDeleteFile(t.Key, f)
			lock.Unlock()
			cleaned++
		}
	}
	log.Notice("Removed %d old files, new size: %d, %d files", cleaned, cache.totalSize, cache.cachedFiles.Count())
	return cleaned > 0
}

// singleClean runs a single clean of the cache. It's split out for testing purposes.
func (cache *Cache) singleClean(lowWaterMark, highWaterMark int64) bool {
	log.Debug("Total size: %d High water mark: %d", cache.totalSize, highWaterMark)
	if cache.totalSize > highWaterMark {
		log.Info("Cleaning cache...")
		files := cache.filesToClean(lowWaterMark)
		log.Info("Identified %d files to clean...", len(files))
		for _, file := range files {
			lock := cache.lockFile(file.path, true, file.file.size)
			cache.removeAndDeleteFile(file.path, file.file)
			lock.Unlock()
		}
		return true
	}
	return false
}

// cachedFilePath embeds a cachedFile but with the path too.
type cachedFilePath struct {
	file *cachedFile
	path string
}

type cachedFilePaths []cachedFilePath

func (c cachedFilePaths) Len() int      { return len(c) }
func (c cachedFilePaths) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c cachedFilePaths) Less(i, j int) bool {
	return c[i].file.lastReadTime.Before(c[j].file.lastReadTime)
}

// filesToClean returns a list of files that should be cleaned, ie. the least interesting
// artifacts in the cache according to some heuristic. Removing all of them will be
// sufficient to reduce the cache size below lowWaterMark.
func (cache *Cache) filesToClean(lowWaterMark int64) cachedFilePaths {
	ret := make(cachedFilePaths, 0, len(cache.cachedFiles))
	for t := range cache.cachedFiles.IterBuffered() {
		ret = append(ret, cachedFilePath{file: t.Val.(*cachedFile), path: t.Key})
	}
	sort.Sort(&ret)

	sizeToDelete := cache.totalSize - lowWaterMark
	var sizeDeleted int64
	for i, file := range ret {
		if sizeDeleted >= sizeToDelete {
			return ret[0:i]
		}
		sizeDeleted += file.file.size
	}
	return ret
}
