package asp

import (
	"fmt"
	"strings"
	"sync"
)

type execKey string
type execValue interface {
	// Found returns true when the value has been found in the cache
	Found() bool
}

var (
	// The output from exec() is memoized by default
	memoizedCmdsLock sync.RWMutex
	memoizedCmds     map[execKey]execValue
)

func init() {
	memoizedCmdsLock.Lock()
	defer memoizedCmdsLock.Unlock()

	const initCacheSize = 16
	memoizedCmds = make(map[execKey]execValue, initCacheSize)
}

// getCachedExecOutput returns the output if found, sets found to true if found,
// and returns a held promise that must be either cancelled or completed.
func getCachedExecOutput(args []string) (output string, found bool, promise execPromisePending) {
	key := makeExecKey(args)

	// We must perform manual locking for this first get test and can't rely on defer
	memoizedCmdsLock.RLock()
	rawVal, found := memoizedCmds[key]
	if val, ok := rawVal.(*execData); found && ok {
		// Simple case, the cmdArgs yielded a cachable output
		memoizedCmdsLock.RUnlock()
		return val.Value(), true, nil
	}
	memoizedCmdsLock.RUnlock()

	// Promote to an exclusive, writer lock
	memoizedCmdsLock.Lock()

	// With an exclusive lock, attempt to see if the cache has been populated.
	rawVal, found = memoizedCmds[key]
	if !found {
		// Create and return a new promise
		promise := &execPromise{lock: &sync.Mutex{}, key: key}
		promise.lock.Lock()
		promise.cv = sync.NewCond(promise.lock)
		memoizedCmds[key] = promise

		memoizedCmdsLock.Unlock()
		return "", false, promise
	}

	// Depending on the data found in the cache, return the value or block on the
	// promise's CV.
	switch val := rawVal.(type) {
	case *execData:
		// Simple case, the cmdArgs yielded a cachable output even though there was
		// a race between dropping the lock above and now.
		memoizedCmdsLock.Unlock()
		return val.Value(), true, nil
	case *execPromise:
		promise := val
		promise.cv.L.Lock()
		for {
			memoizedCmdsLock.Unlock()
			promise.cv.Wait()
			memoizedCmdsLock.Lock()
			rawVal, found = memoizedCmds[key]
			if !found {
				// The promise lock that we're holding is no longer associated with a
				// key in the memoization map (likely because the previous promise ran
				// and exited unsuccessfully).  Create a promise and hope for the best
				// on the second invocation..
				promise := &execPromise{lock: &sync.Mutex{}, key: key}
				promise.lock.Lock()
				promise.cv = sync.NewCond(promise.lock)
				memoizedCmds[key] = promise
				return "", false, promise
			}

			if data, ok := rawVal.(*execData); ok {
				promise.cv.L.Unlock()
				memoizedCmdsLock.Unlock()
				return data.Value(), true, nil
			} else {
				panic(fmt.Sprintf("unknown type in execValue: %T", val))
			}
		}
	default:
		panic(fmt.Sprintf("unknown type in execValue: %T", val))
	}
}

// execData satisfies the execValue interface and may be assigned to
// memoizedCmds.
type execData struct {
	value string
}

// Found unconditionally returns true for execData
func (d *execData) Found() bool {
	return true
}

func (d *execData) Value() string {
	return d.value
}

// execPromise satisfies the execValue interface and may be assigned to
// memoizedCmds.
type execPromise struct {
	// lock protects the entire execValue
	lock *sync.Mutex

	// cv is used to indicate that a lookup is in progress.
	cv *sync.Cond

	// key is a copy of the cached memoizedCmds key
	key execKey
}

// Found unconditionally returns false for execData
func (p *execPromise) Found() bool {
	return false
}

// cancel causes the execPromise to wake up and signal one other blocking thread
// to wake up and proceed. Cancel can only be called with the p.lock held (the
// default from getCachedExecOutput()).
func (p *execPromise) cancel() {
	defer p.cv.L.Unlock()

	p.cv.Signal()
}

// complete sets a value for the promise and broadcasts to all blocked go
// routines to wake and return a found value.  Complete must succeed.  Complete
// can only be called with the p.lock held (the default from
// getCachedExecOutput()).  cancel() must be used to unlock the Locker held by
// the cv.
func (p *execPromise) complete(out []byte) {
	memoizedCmdsLock.Lock()
	defer memoizedCmdsLock.Unlock()

	memoizedCmds[p.key] = &execData{value: string(out)}
}

// execPromisePending is the value returned when a promise needs to be completed
// (or cancelled).
type execPromisePending interface {
	cancel()
	complete(out []byte)
}

// makeExecKey returns an execKey
func makeExecKey(args []string) execKey {
	return execKey(strings.Join(args, ""))
}
