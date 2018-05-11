package fs

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// boolTrueHashValue is used when we need to write something indicating a bool in the input.
var boolTrueHashValue = []byte{2}

// A PathHasher is responsible for hashing & remembering paths.
type PathHasher struct {
	memo  map[string][]byte
	mutex sync.RWMutex
	root  string
}

// NewPathHasher returns a new PathHasher based on the given root directory.
func NewPathHasher(root string) *PathHasher {
	return &PathHasher{
		memo: map[string][]byte{},
		root: root,
	}
}

// Hash hashes a single path.
// It is memoised and so will only hash each path once, unless recalc is true which will
// then force a recalculation of it.
func (hasher *PathHasher) Hash(path string, recalc bool) ([]byte, error) {
	path = hasher.ensureRelative(path)
	if !recalc {
		hasher.mutex.RLock()
		cached, present := hasher.memo[path]
		hasher.mutex.RUnlock()
		if present {
			return cached, nil
		}
	}
	result, err := hasher.hash(path)
	if err == nil {
		hasher.mutex.Lock()
		hasher.memo[path] = result
		hasher.mutex.Unlock()
	}
	return result, err
}

// MustHash is as Hash but panics on error.
func (hasher *PathHasher) MustHash(path string) []byte {
	hash, err := hasher.Hash(path, false)
	if err != nil {
		panic(err)
	}
	return hash
}

// MoveHash is used when we move files from tmp to out and there was one there before; that's
// the only case in which the hash of a filepath could change.
func (hasher *PathHasher) MoveHash(oldPath, newPath string, copy bool) {
	oldPath = hasher.ensureRelative(oldPath)
	newPath = hasher.ensureRelative(newPath)
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()
	if oldHash, present := hasher.memo[oldPath]; present {
		hasher.memo[newPath] = oldHash
		// If the path is in plz-out/tmp we aren't ever going to use it again, so free some space.
		if !copy && strings.HasPrefix(oldPath, "plz-out/tmp") {
			delete(hasher.memo, oldPath)
		}
	}
}

// SetHash is used to directly set a hash for a path.
// This is used for remote files where we download them & therefore know the hash as they come in.
// TODO(peterebden): We should probably use this more for things like caches and so forth...
func (hasher *PathHasher) SetHash(path string, hash []byte) {
	hasher.mutex.Lock()
	hasher.memo[path] = hash
	hasher.mutex.Unlock()
}

func (hasher *PathHasher) hash(path string) ([]byte, error) {
	h := sha1.New()
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		// Handle symlinks specially (don't attempt to read their contents).
		dest, err := os.Readlink(path)
		if err != nil {
			return nil, err
		}
		// Write something arbitrary indicating this is a symlink.
		// This isn't quite perfect - it could potentially get mixed up with a file with the
		// appropriate contents, but that is not really likely.
		h.Write(boolTrueHashValue)
		if rel := hasher.ensureRelative(dest); rel != dest || !filepath.IsAbs(dest) {
			// Inside the root of our repo so it's something we manage - just hash its (relative) destination
			h.Write([]byte(rel))
		} else {
			// Outside the repo; it's a system tool, so we hash its contents.
			err := hasher.fileHash(h, dest)
			return h.Sum(nil), err
		}
		return h.Sum(nil), nil
	} else if err == nil && info.IsDir() {
		err = WalkMode(path, func(p string, isDir bool, mode os.FileMode) error {
			if mode&os.ModeSymlink != 0 {
				// Is a symlink, must verify that it's not a link outside the tmp dir.
				deref, err := filepath.EvalSymlinks(p)
				if err != nil {
					return err
				}
				if !strings.HasPrefix(deref, path) {
					return fmt.Errorf("Output %s links outside the build dir (to %s)", p, deref)
				}
				// Deliberately do not attempt to read it. We will read the contents later since
				// it is a link within the temp dir anyway, and if it's a link to a directory
				// it can introduce a cycle.
				// Just write something to the hash indicating that we found something here,
				// otherwise rules might be marked as unchanged if they added additional symlinks.
				h.Write(boolTrueHashValue)
			} else if !isDir {
				return hasher.fileHash(h, p)
			}
			return nil
		})
	} else {
		err = hasher.fileHash(h, path) // let this handle any other errors
	}
	return h.Sum(nil), err
}

// Calculate the hash of a single file
func (hasher *PathHasher) fileHash(h hash.Hash, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(h, file)
	file.Close()
	return err
}

// ensureRelative ensures a path is relative to the repo root.
// This is important for getting best performance from memoizing the path hashes.
func (hasher *PathHasher) ensureRelative(path string) string {
	if strings.HasPrefix(path, hasher.root) {
		return strings.TrimLeft(strings.TrimPrefix(path, hasher.root), "/")
	}
	return path
}
