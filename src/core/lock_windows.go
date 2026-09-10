package core

import (
	"os"
	"sync"

	"golang.org/x/sys/windows"
)

// These mirror the flock(2) constants; their values are arbitrary since Windows doesn't
// define them, but they must remain distinct bits because callers combine and test them.
const (
	lockShared      = 0x1
	lockExclusive   = 0x2
	lockUnlock      = 0x8
	lockNonBlocking = 0x4
)

// LockFileEx locks a byte range rather than a whole file. We lock a single byte far past any
// plausible content so that the PID written into the lock file stays readable by other
// processes, which is what produces the "process N has already acquired the lock" message.
const (
	lockOffsetLow  = 0
	lockOffsetHigh = 0x40000000
)

// Windows has no equivalent of flock's atomic conversion between shared and exclusive on a
// single handle, so we have to release before re-acquiring. Track what each handle holds in
// order to do that only when it's actually needed.
var (
	locksMux sync.Mutex
	locks    = map[*os.File]bool{}
)

// flock applies or releases an advisory lock on an open file.
//
// N.B. unlike flock(2), changing mode on a handle that already holds a lock is not atomic:
// the lock is dropped and re-taken, so another process can take it in between. Please only
// changes mode at startup (shared -> exclusive in acquireRepoLock), so in practice this
// window is not contended.
func flock(file *os.File, how int) error {
	handle := windows.Handle(file.Fd())
	overlapped := &windows.Overlapped{Offset: lockOffsetLow, OffsetHigh: lockOffsetHigh}

	locksMux.Lock()
	defer locksMux.Unlock()

	if how&lockUnlock != 0 {
		if !locks[file] {
			return nil
		}
		delete(locks, file)
		return windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
	}

	if locks[file] {
		if err := windows.UnlockFileEx(handle, 0, 1, 0, overlapped); err != nil {
			return err
		}
		delete(locks, file)
	}

	var flags uint32
	if how&lockExclusive != 0 {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	if how&lockNonBlocking != 0 {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	if err := windows.LockFileEx(handle, flags, 0, 1, 0, overlapped); err != nil {
		return err
	}
	locks[file] = true
	return nil
}
