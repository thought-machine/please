// The logic below relies heavily on flock (advisory locks).

package core

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/thought-machine/please/src/fs"
)

const repoLockFilePath = "plz-out/.lock"

const cachedirTagFile = "plz-out/CACHEDIR.TAG"

const cachedirTagFileContents = `Signature: 8a477f597d28d172789f06886806bc55
# This file is a cache directory tag created by Please.
# For information about cache directory tags see https://bford.info/cachedir/
`

var repoLockFile *os.File

// AcquireSharedRepoLock acquires a shared lock on the repo lock file. The file descriptor is reused if already opened
// allowing its lock mode to be replaced. Dies if the lock cannot be successfully acquired.
func AcquireSharedRepoLock() {
	if err := acquireRepoLock(syscall.LOCK_SH, log.Warning); err != nil {
		log.Fatal(err)
	}
}

// AcquireExclusiveRepoLock acquires an exclusive lock on the repo lock file. The file descriptor is reused if already opened
// allowing its lock mode to be replaced. Dies if the lock cannot be successfully acquired.
func AcquireExclusiveRepoLock() {
	if err := acquireRepoLock(syscall.LOCK_EX, log.Warning); err != nil {
		log.Fatal(err)
	}
}

// AcquireExclusiveRepoLockQuietly is like AcquireExclusiveRepoLock but log messages are not shown if we need to wait for another process.
func AcquireExclusiveRepoLockQuietly() {
	if err := acquireRepoLock(syscall.LOCK_EX, log.Info); err != nil {
		log.Fatal(err)
	}
}

// ReleaseRepoLock releases any lock mode on the repo lock file.
func ReleaseRepoLock() {
	ReleaseFileLock(repoLockFile)
	repoLockFile = nil
}

// Base function that allows to set up different repo lock modes and facilitate testing.
func acquireRepoLock(how int, levelLog logFunc) error {
	if err := openRepoLockFile(); err != nil {
		return err
	} else if err := acquireFileLock(repoLockFile, how, levelLog); err != nil {
		return err
	}
	// Write a cachedir file that indicates to some tools that this is non-essential and not to back up.
	if err := os.WriteFile(cachedirTagFile, []byte(cachedirTagFileContents), 0644); err != nil {
		log.Warningf("Failed to write cachedir tag file: %s", err)
	}
	return nil
}

// This acts like a singleton allowing the same file descriptor to used to override a previously set lock
// (from shared to exclusive and vice-versa) within the same process.
func openRepoLockFile() error {
	// Already opened
	if repoLockFile != nil {
		return nil
	}

	var err error
	if repoLockFile, err = openLockFile(repoLockFilePath); err != nil {
		return err
	}
	return nil
}

// AcquireExclusiveFileLock opens a file to acquire an exclusive lock.
// Dies if the lock cannot be successfully acquired.
func AcquireExclusiveFileLock(filePath string) *os.File {
	lockFile, err := acquireOpenFileLock(filePath, syscall.LOCK_EX)
	if err != nil {
		log.Fatal(err)
	}
	return lockFile
}

// Base function that allows to set up different lock modes and facilitate testing.
func acquireOpenFileLock(filePath string, how int) (*os.File, error) {
	lockFile, err := openLockFile(filePath)
	if err != nil {
		return nil, err
	}

	if err = acquireFileLock(lockFile, how, log.Debug); err != nil {
		return nil, err
	}

	return lockFile, nil
}

// ReleaseFileLock releases the lock and closes the file handle.
// Does not die on errors, at this point it wouldn't really do any good.
func ReleaseFileLock(file *os.File) {
	if file == nil {
		return
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		log.Errorf("Failed to release lock for %s: %s", file.Name(), err) // No point making this fatal really
	}
	if err := file.Close(); err != nil {
		log.Errorf("Failed to close lock file %s: %s", file.Name(), err)
	}
}

type logFunc func(format string, args ...interface{})

func acquireFileLock(file *os.File, how int, levelLog logFunc) error {
	// Try a non-blocking acquire first so we can warn the user if we're waiting.
	log.Debug("Attempting to acquire lock for %s...", file.Name())
	err := syscall.Flock(int(file.Fd()), how|syscall.LOCK_NB)
	if err != nil {
		pid, err := os.ReadFile(file.Name())
		if err == nil && len(pid) > 0 {
			levelLog("Looks like process with PID %s has already acquired the lock for %s. Waiting for it to finish...", string(pid), file.Name())
		} else {
			levelLog("Looks like another process has already acquired the lock for %s. Waiting for it to finish...", file.Name())
		}

		if err := syscall.Flock(int(file.Fd()), how); err != nil {
			return fmt.Errorf("Failed to acquire lock for %s: %w", file.Name(), err)
		}
	}
	log.Debug("Acquired lock for %s", file.Name())

	// Record content
	if err := file.Truncate(0); err == nil {
		file.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0)
	}

	return nil
}

// Try to open a file for locking
func openLockFile(filePath string) (*os.File, error) {
	lockFile, err := fs.OpenDirFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open %s to acquire lock: %w", filePath, err)
	}
	return lockFile, nil
}
