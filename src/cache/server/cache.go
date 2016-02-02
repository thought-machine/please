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

// Handles synchronisation of storing/retrieving artifacts.
var cachedFiles = map[string]*cachedFile{}
var mutex sync.RWMutex
var totalSize int64

var cachePath = ".plz-cache"

// Init initialises the cache and fires off a background cleaner goroutine which runs every
// cleanFrequency seconds. The high and low water marks control a (soft) max size and a (harder)
// minimum size.
// This causes an initial scan of the directory and is needed to retrieve any preexisting artifacts.
// If it's not called the cache will still function but won't know about anything existing beforehand.
func Init(path string, cleanFrequency int, lowWaterMark, highWaterMark int64) {
	log.Notice("Initialising cache with settings:\n  Path: %s\n  Clean frequency: %d\n  Low water mark: %d\n  High water mark: %d",
		path, cleanFrequency, lowWaterMark, highWaterMark)
	cachePath = path
	scan(path)
	go clean(path, cleanFrequency, lowWaterMark, highWaterMark)
}

// scan scans the directory tree for files.
func scan(path string) {
	mutex.Lock()
	defer mutex.Unlock()
	cachedFiles = map[string]*cachedFile{}
	totalSize = 0

	if !core.PathExists(path) {
		if err := os.MkdirAll(path, core.DirPermissions); err != nil {
			log.Fatalf("Failed to create cache directory %s: %s", path, err)
		}
		return
	}

	log.Info("Scanning cache directory %s...", path)
	filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("%s", err)
		} else if !info.IsDir() { // We don't have directory entries.
			name = name[len(path)+1 : len(name)]
			log.Debug("Found file %s", name)
			size := info.Size()
			cachedFiles[name] = &cachedFile{
				lastReadTime: time.Now(),
				readCount:    0,
				size:         size,
			}
			totalSize += size
		}
		return nil
	})
	cachePath = path
	log.Info("Scan complete")
}

// lockFile locks a file for reading or writing.
// It returns a locked mutex corresponding to that file or nil if there is none.
// The caller should .Unlock() the mutex once they're done with it.
func lockFile(path string, write bool, size int64) *cachedFile {
	mutex.Lock()
	defer mutex.Unlock()
	file, present := cachedFiles[path]
	if !present {
		// If we're writing we insert a new one, if we're reading we don't.
		if !write {
			return nil
		}
		file = &cachedFile{
			lastReadTime: time.Now(),
			readCount:    0,
			size:         size,
		}
		cachedFiles[path] = file
		totalSize += size
	}
	if write {
		file.Lock()
	} else {
		file.RLock()
		file.readCount++
	}
	return file
}

// removeFile deletes a file from the cache map. It does not remove the on-disk file.
// Caller should already hold the mutex before calling this
func removeFile(path string, file *cachedFile) {
	delete(cachedFiles, path)
	totalSize -= file.size
}

// RetrieveArtifact takes in the artifact path as a parameter and checks in the base server
// file directory to see if the file exists in the given path. If found, the function will
// return whatever's been stored there, which might be a directory and therefore contain
// multiple files to be returned.
func RetrieveArtifact(artPath string) (map[string][]byte, error) {
	lock := lockFile(artPath, false, 0)
	if lock == nil {
		return nil, os.ErrNotExist
	}
	defer lock.RUnlock()

	fullPath := path.Join(cachePath, artPath)
	ret := map[string][]byte{}
	if err := filepath.Walk(fullPath, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !info.IsDir() {
			body, err := ioutil.ReadFile(name)
			if err != nil {
				return err
			}
			ret[name[len(cachePath)+1:len(name)]] = body
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

// StoreArtifact takes in the artifact content and path as parameters and creates a file with
// the given content in the given path.
// The function will return the first error found in the process, or nil if the process is successful.
func StoreArtifact(artPath string, key []byte) error {
	log.Info("Storing artifact %s", artPath)
	lock := lockFile(artPath, true, int64(len(key)))
	defer lock.Unlock()

	artPath = path.Join(cachePath, artPath)
	dirPath := path.Dir(artPath)
	if err := os.MkdirAll(dirPath, core.DirPermissions); err != nil {
		log.Warning("Couldn't create path %s in http cache: %s", dirPath, err)
		return err
	}
	log.Debug("Writing artifact to %s", artPath)
	if err := core.WriteFile(bytes.NewReader(key), artPath, 0); err != nil {
		log.Error("Could not create %s artifact: %s", artPath, err)
		return err
	}
	return nil
}

// DeleteArtifact takes in the artifact path as a parameter and removes the artifact from disk.
// The function will return the first error found in the process, or nil if the process is successful.
func DeleteArtifact(artPath string) error {
	// We need to search the entire map for prefixes. Pessimism follows...
	mutex.Lock()
	for p, file := range cachedFiles {
		if strings.HasPrefix(p, artPath) {
			file.Lock()
			removeFile(path.Join(cachePath, p), file)
			file.Unlock()
		}
	}
	mutex.Unlock()
	return os.RemoveAll(path.Join(cachePath, artPath))
}

// DeleteAllArtifacts will remove all files in the cache directory.
// The function will return the first error found in the process, or nil if the process is successful.
func DeleteAllArtifacts() error {
	// Empty entire cache now.
	mutex.Lock()
	cachedFiles = map[string]*cachedFile{}
	mutex.Unlock()
	defer scan(cachePath) // Rescan whatever's left afterwards.

	files, err := ioutil.ReadDir(cachePath)
	if err != nil && os.IsNotExist(err) {
		log.Warning("%s directory does not exist, nothing to delete.", cachePath)
		return nil
	} else if err != nil {
		log.Error("Error reading cache directory: %s", err)
		return err
	}
	for _, file := range files {
		p := path.Join(cachePath, file.Name())
		if err := os.RemoveAll(p); err != nil {
			log.Error("Failed to remove directory %s: %s", p, err)
			return err
		}
	}
	return nil
}

// clean implements a periodic clean of the cache to remove old artifacts.
func clean(path string, cleanFrequency int, lowWaterMark, highWaterMark int64) {
	for _ = range time.NewTicker(time.Duration(cleanFrequency) * time.Second).C {
		if totalSize > highWaterMark {
			log.Info("Cleaning cache...")
			files := filesToClean(lowWaterMark)
			log.Info("Identified %d files to clean...", len(files))
			for _, file := range files {
				removeFile(file.path, file.file)
				if err := os.RemoveAll(file.path); err != nil {
					log.Error("Failed to remove artifact: %s", err)
				}
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
func filesToClean(lowWaterMark int64) cachedFilePaths {
	mutex.Lock()
	ret := make(cachedFilePaths, 0, len(cachedFiles))
	for path, file := range cachedFiles {
		ret = append(ret, cachedFilePath{file: file, path: path})
	}
	mutex.Unlock()
	sort.Sort(&ret)

	sizeToDelete := totalSize - lowWaterMark
	var sizeDeleted int64
	for i, file := range ret {
		if sizeDeleted >= sizeToDelete {
			return ret[0:i]
		}
		sizeDeleted += file.file.size
	}
	return ret
}
