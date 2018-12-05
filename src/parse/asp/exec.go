package asp

import (
	"strconv"
	"strings"
	"sync"
)

type execKey string
type execPromise struct {
	cv        *sync.Cond
	out       string
	cancelled bool
	finished  bool
}

var (
	// The output from exec() is memoized by default
	execCacheLock  sync.RWMutex
	execCachedOuts map[execKey]string

	execPromisesLock sync.Mutex
	execPromises     map[execKey]*execPromise
)

func init() {
	execCacheLock.Lock()
	defer execCacheLock.Unlock()

	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()

	const initCacheSize = 16
	execCachedOuts = make(map[execKey]string, initCacheSize)
	execPromises = make(map[execKey]*execPromise, initCacheSize)
}

// execCancelPromise cancels any pending promises
func execCancelPromise(key execKey, args []string) {
	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()
	if promise, found := execPromises[key]; found {
		delete(execPromises, key)
		promise.cv.L.Lock()
		promise.cancelled = true
		promise.cv.Broadcast()
		promise.cv.L.Unlock()
	}
}

// execGetCachedOutput returns the output if found, sets found to true if found,
// and returns a held promise that must be either cancelled or completed.
func execGetCachedOutput(key execKey, args []string) (output string, found bool) {
	execCacheLock.RLock()
	out, found := execCachedOuts[key]
	execCacheLock.RUnlock()
	if found {
		return out, true
	}

	// Re-check with exclusive lock held
	execCacheLock.Lock()
	out, found = execCachedOuts[key]
	if found {
		execCacheLock.Unlock()
		return out, true
	}

	execPromisesLock.Lock()
	promise, found := execPromises[key]
	if !found {
		promise = &execPromise{
			cv: sync.NewCond(&sync.Mutex{}),
		}
		execPromises[key] = promise

		execCacheLock.Unlock()
		execPromisesLock.Unlock()
		return "", false // Let the caller fulfill the promise
	}
	execCacheLock.Unlock() // Release now that we've recorded our promise

	promise.cv.L.Lock() // Lock our promise before we unlock execPromisesLock
	execPromisesLock.Unlock()

	for {
		switch {
		case promise.finished:
			promise.cv.L.Unlock()
			return promise.out, true
		case promise.cancelled:
			return "", false
		default:
			promise.cv.Wait()
		}
	}
}

// execMakeKey returns an execKey
func execMakeKey(s *scope, args []string, wantStdout bool, wantStderr bool) execKey {
	// TODO: Use scope to construct a better cache key when looking up cached
	// outputs.
	keyArgs := make([]string, 0, len(args)+2)
	keyArgs = append(keyArgs, args...)
	keyArgs = append(keyArgs, strconv.FormatBool(wantStdout))
	keyArgs = append(keyArgs, strconv.FormatBool(wantStderr))

	return execKey(strings.Join(keyArgs, ""))
}

// execSetCachedOutput sets a value to be cached
func execSetCachedOutput(key execKey, args []string, output string) {
	execCacheLock.Lock()
	execCachedOuts[key] = output
	execCacheLock.Unlock()

	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()
	if promise, found := execPromises[key]; found {
		delete(execPromises, key)
		promise.cv.L.Lock()
		promise.out = output
		promise.finished = true
		promise.cv.Broadcast()
		promise.cv.L.Unlock()
	}
}
