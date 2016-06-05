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
	"time"

	"cache/tools"
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
	cachedFiles map[string]*cachedFile
	mutex       sync.RWMutex
	totalSize   int64
	rootPath    string
}

// NewCache initialises the cache and fires off a background cleaner goroutine which runs every
// cleanFrequency seconds. The high and low water marks control a (soft) max size and a (harder)
// minimum size.
func NewCache(path string, cleanFrequency int, lowWaterMark, highWaterMark int64) *Cache {
	log.Notice("Initialising cache with settings:\n  Path: %s\n  Clean frequency: %d\n  Low water mark: %d\n  High water mark: %d",
		path, cleanFrequency, lowWaterMark, highWaterMark)
	cache := newCache(path)
	go cache.clean(cleanFrequency, lowWaterMark, highWaterMark)
	return cache
}

// newCache is an internal constructor intended mostly for testing. It doesn't start the cleaner goroutine.
func newCache(path string) *Cache {
	cache := &Cache{rootPath: path}
	cache.scan()
	return cache
}

// scan scans the directory tree for files.
func (cache *Cache) scan() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.cachedFiles = map[string]*cachedFile{}
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
			name = name[len(cache.rootPath)+1 : len(name)]
			log.Debug("Found file %s", name)
			size := info.Size()
			cache.cachedFiles[name] = &cachedFile{
				lastReadTime: time.Unix(tools.AccessTime(info), 0),
				readCount:    0,
				size:         size,
			}
			cache.totalSize += size
		}
		return nil
	})
	log.Info("Scan complete, found %d entries", len(cache.cachedFiles))
}

// lockFile locks a file for reading or writing.
// It returns a locked mutex corresponding to that file or nil if there is none.
// The caller should .Unlock() the mutex once they're done with it.
func (cache *Cache) lockFile(path string, write bool, size int64) *cachedFile {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	file, present := cache.cachedFiles[path]
	if !present {
		// If we're writing we insert a new one, if we're reading we don't.
		if !write {
			return nil
		}
		file = &cachedFile{
			readCount: 0,
			size:      size,
		}
		cache.cachedFiles[path] = file
		cache.totalSize += size
	}
	if write {
		file.Lock()
	} else {
		file.RLock()
		file.readCount++
	}
	file.lastReadTime = time.Now()
	return file
}

// removeFile deletes a file from the cache map. It does not remove the on-disk file.
// Caller should already hold the mutex before calling this
func (cache *Cache) removeFile(path string, file *cachedFile) {
	delete(cache.cachedFiles, path)
	cache.totalSize -= file.size
	log.Debug("Removing file %s, saves %d, new size will be %d", path, file.size, cache.totalSize)
}

// removeAndDeleteFile deletes a file from the cache map and on-disk.
// Caller should already hold the mutex before calling this
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

	ret := map[string][]byte{}
	if err := filepath.Walk(fullPath, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !info.IsDir() {
			body, err := ioutil.ReadFile(name)
			if err != nil {
				return err
			}
			ret[name[len(cache.rootPath)+1:len(name)]] = body
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
	// We need to search the entire map for prefixes. Pessimism follows...
	cache.mutex.Lock()
	for p, file := range cache.cachedFiles {
		if strings.HasPrefix(p, artPath) {
			file.Lock()
			cache.removeFile(p, file)
			file.Unlock()
		}
	}
	cache.mutex.Unlock()
	return os.RemoveAll(path.Join(cache.rootPath, artPath))
}

// DeleteAllArtifacts will remove all files in the cache directory.
// The function will return the first error found in the process, or nil if the process is successful.
func (cache *Cache) DeleteAllArtifacts() error {
	// Empty entire cache now.
	cache.mutex.Lock()
	cache.cachedFiles = map[string]*cachedFile{}
	cache.mutex.Unlock()
	defer cache.scan() // Rescan whatever's left afterwards.

	files, err := ioutil.ReadDir(cache.rootPath)
	if err != nil && os.IsNotExist(err) {
		log.Warning("%s directory does not exist, nothing to delete.", cache.rootPath)
		return nil
	} else if err != nil {
		log.Errorf("Error reading cache directory: %s", err)
		return err
	}
	for _, file := range files {
		p := path.Join(cache.rootPath, file.Name())
		if err := os.RemoveAll(p); err != nil {
			log.Errorf("Failed to remove directory %s: %s", p, err)
			return err
		}
	}
	return nil
}

// clean implements a periodic clean of the cache to remove old artifacts.
func (cache *Cache) clean(cleanFrequency int, lowWaterMark, highWaterMark int64) {
	for _ = range time.NewTicker(time.Duration(cleanFrequency) * time.Second).C {
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
		}
	}
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
	cache.mutex.Lock()
	ret := make(cachedFilePaths, 0, len(cache.cachedFiles))
	for path, file := range cache.cachedFiles {
		ret = append(ret, cachedFilePath{file: file, path: path})
	}
	cache.mutex.Unlock()
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
