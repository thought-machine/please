package fs

import (
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/xattr"
)

// symlinkHashValue is used when we need to write something arbitrary indicating the input is a symlink.
var symlinkHashValue = []byte{2}

// A PathHasher is responsible for hashing & remembering paths.
type PathHasher struct {
	new       func() hash.Hash
	memo      map[string][]byte
	wait      map[string]*pendingHash
	tasks     chan hashTask
	mutex     sync.RWMutex
	root      string
	xattrName string
	useXattrs bool
	algo      string
}

type pendingHash struct {
	Ch   chan struct{}
	Hash []byte
	Err  error
}

type hashTask struct {
	Path string
	Ch   chan hashResult
}

type hashResult struct {
	Hash []byte
	Err  error
}

// NewPathHasher returns a new PathHasher based on the given root directory.
// parallelism controls the maximum number of concurrent hash operations allowed
func NewPathHasher(root string, useXattrs bool, hash func() hash.Hash, algo string, parallelism int) *PathHasher {
	var hashSuffix string
	if algo != "sha1" {
		hashSuffix = fmt.Sprintf("_%s", algo)
	}
	h := &PathHasher{
		new:       hash,
		memo:      map[string][]byte{},
		wait:      map[string]*pendingHash{},
		root:      root,
		useXattrs: useXattrs,
		xattrName: "user.plz_hash" + hashSuffix,
		algo:      algo,
	}
	if parallelism <= 1 {
		return h
	}
	// This can be bigger than our available parallelism to allow queueing up future tasks.
	h.tasks = make(chan hashTask, 10 * parallelism)
	for i := 0; i < parallelism; i++ {
		go h.runTask()
	}
	return h
}

// Size returns the size of the hash this hasher will return, in bytes.
func (hasher *PathHasher) Size() int {
	return hasher.new().Size()
}

// NewHash returns a new hash.Hash instance from this hasher.
func (hasher *PathHasher) NewHash() hash.Hash {
	return hasher.new()
}

// DisableXattrs turns off xattr support, which bypasses using them to record file hashes.
func (hasher *PathHasher) DisableXattrs() {
	hasher.useXattrs = false
}

// AlgoName returns the name of the algorithm. Used to aid with better error messages.
func (hasher *PathHasher) AlgoName() string {
	return hasher.algo
}

// Hash hashes a single path.
// It is memoised and so will only hash each path once, unless recalc is true which will
// then force a recalculation of it.
// If store is true then the hash may be stored permanently; this should not be set for files that
// are user-controlled.
// TODO(peterebden): ensure that xattrs are marked correctly on cache retrieval.
func (hasher *PathHasher) Hash(path string, recalc, store bool) ([]byte, error) {
	path = hasher.ensureRelative(path)
	if !recalc {
		hasher.mutex.RLock()
		cached, present := hasher.memo[path]
		hasher.mutex.RUnlock()
		if present {
			return cached, nil
		}
	}
	// This check is important; if the file doesn't exist now, we don't want that
	// recorded forever in hasher.wait since it might get created later.
	if !PathExists(path) {
		return nil, fmt.Errorf("cannot calculate hash for %s: %s", path, os.ErrNotExist)
	}
	hasher.mutex.Lock()
	if pending, present := hasher.wait[path]; present {
		hasher.mutex.Unlock()
		<-pending.Ch
		return pending.Hash, pending.Err
	}
	pending := &pendingHash{Ch: make(chan struct{})}
	hasher.wait[path] = pending
	hasher.mutex.Unlock()
	result, err := hasher.hash(path, store, !recalc)
	hasher.mutex.Lock()
	if err == nil {
		hasher.memo[path] = result
	}
	delete(hasher.wait, path)
	hasher.mutex.Unlock()
	pending.Hash = result
	pending.Err = err
	close(pending.Ch)
	return result, err
}

// MustHash is as Hash but panics on error.
func (hasher *PathHasher) MustHash(path string) []byte {
	hash, err := hasher.Hash(path, false, false)
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
	hasher.storeHash(path, hash)
}

func (hasher *PathHasher) hash(path string, store, read bool) ([]byte, error) {
	// Try to read xattrs first so we don't have to hash the whole thing.
	if read && strings.HasPrefix(path, "plz-out/") && hasher.useXattrs {
		if b, err := xattr.LGet(path, hasher.xattrName); err == nil {
			return b, nil
		}
	}
	h := hasher.new()
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
		h.Write(symlinkHashValue)
		if rel := hasher.ensureRelative(dest); (rel != dest || !filepath.IsAbs(dest)) && !filepath.IsAbs(path) {
			// Inside the root of our repo so it's something we manage - just hash its (relative) destination
			h.Write([]byte(rel))
		} else {
			// Outside the repo; it's a system tool, so we hash its contents.
			return hasher.fileHash(path)
		}
		return h.Sum(nil), nil
	} else if err == nil {
		err = hasher.hashPath(h, path, info.IsDir())
	}
	hash := h.Sum(nil)
	if err != nil {
		return hash, err
	} else if store && hasher.useXattrs {
		hasher.storeHash(path, hash)
	}
	return hash, err
}

// hashPath hashes a single path, which might be a directory.
func (hasher *PathHasher) hashPath(h hash.Hash, path string, isDir bool) error {
	if !isDir {
		return hasher.inPlaceFileHash(h, path)
	} else if hasher.tasks == nil {
		// Serial implementation
		return WalkMode(path, func(p string, mode Mode) error {
			if mode.IsSymlink() {
				// Deliberately do not attempt to read it. We will read the contents later since
				// it is a link within the temp dir anyway, and if it's a link to a directory
				// it can introduce a cycle.
				// Just write something to the hash indicating that we found something here,
				// otherwise rules might be marked as unchanged if they added additional symlinks.
				h.Write(symlinkHashValue)
			} else if !mode.IsDir() {
				return hasher.inPlaceFileHash(h, p)
			}
			return nil
		})
	}
	tasks := []hashTask{}
	if err := WalkMode(path, func(p string, mode Mode) error {
		if mode.IsDir() {
			return nil
		}
		task := hashTask{Path: p, Ch: make(chan hashResult, 1)}
		tasks = append(tasks, task)
		if mode.IsSymlink() {
			// Short-circuit this since all symlinks have a known value.
			go func() {
				task.Ch <- hashResult{Hash: symlinkHashValue}
			}()
		} else {
			hasher.tasks <- task
		}
		return nil
	}); err != nil {
		return err
	}
	for _, task := range tasks {
		result := <-task.Ch
		if result.Err != nil {
			return result.Err
		}
		h.Write(result.Hash)
	}
	return nil
}

// storeHash stores the hash of a file on it as an xattr.
// This is best-effort since if it fails we can always fall back to a slower but reliable rehash.
func (hasher *PathHasher) storeHash(path string, hash []byte) {
	// Only ever store hashes on output files.
	if !strings.HasPrefix(path, "plz-out/") {
		return
	}
	if err := xattr.LSet(path, hasher.xattrName, hash); err != nil && os.IsPermission(err) {
		// If we get a permission denied, that may be because the output file was readonly.
		// Cheekily attempt to chmod it into submission.
		if info, err := os.Lstat(path); err == nil {
			if err := os.Chmod(path, info.Mode()|0220); err == nil {
				xattr.LSet(path, hasher.xattrName, hash)
				os.Chmod(path, info.Mode())
			}
		}
	}
}

// Calculate the hash of a single file
func (hasher *PathHasher) fileHash(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	h := hasher.NewHash()
	_, err = io.Copy(h, file)
	return h.Sum(nil), err
}

// Calculate the hash of a single file into the given hash
func (hasher *PathHasher) inPlaceFileHash(h hash.Hash, filename string) error {
	sum, err := hasher.fileHash(filename)
	h.Write(sum)
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

// runTask runs one worker goroutine to calculate hashes.
func (hasher *PathHasher) runTask() {
	for task := range hasher.tasks {
		h, err := hasher.fileHash(task.Path)
		task.Ch <- hashResult{Hash: h, Err: err}
	}
}
