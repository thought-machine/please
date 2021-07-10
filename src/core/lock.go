// Contains utility functions for managing an exclusive lock file.
// Based on flock() underneath so

package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/pkg/xattr"
)

const repoLockFilePath = "plz-out/.lock"

type fdMap struct {
	files map[string]*os.File
	mutex sync.RWMutex
}

func newFdMap() fdMap {
	return fdMap{
		files: make(map[string]*os.File),
	}
}

var lockFiles = newFdMap()

func AcquireRepoLock() {
	AcquireFileLock(repoLockFilePath)
}

// AcquireFileLock opens a file and acquires the lock.
// Dies if the lock cannot be successfully acquired.
func AcquireFileLock(filePath string) {
	lockFiles.mutex.Lock()
	defer lockFiles.mutex.Unlock()

	// There is of course technically a bit of a race condition between the file & flock operations here.
	lockFile := openLockFile(filePath)
	// Try a non-blocking acquire first so we can warn the user if we're waiting.
	log.Debug("Attempting to acquire lock %s...", filePath)
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		log.Debug("Acquired lock %s", filePath)
	} else {
		//log.Warning("Looks like another thread has already acquired the lock for %s. Waiting for it to finish...", filePath)
		if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
			log.Fatalf("Failed to acquire lock: %s", err)
		}
	}

	lockFiles.files[filePath] = lockFile

	// Record the operation performed.
	if _, err := lockFile.Seek(0, io.SeekStart); err == nil {
		if n, err := lockFile.Write([]byte(fmt.Sprint(os.Getpid(), "\n"))); err == nil {
			lockFile.Truncate(int64(n))
		}
	}
}

func CheckXattrsSupported(state *BuildState) {
	// Quick test of xattrs; we don't keep trying to use them if they fail here.
	if state.XattrsSupported {
		// This creates the lockfile if it doesn't exist
		openLockFile(repoLockFilePath)
		if err := xattr.Set(repoLockFilePath, "user.plz_build", []byte("lock")); err != nil {
			log.Warning("xattrs are not supported on this filesystem, using fallbacks")
			state.DisableXattrs()
		}
	}
}

// ReleaseFileLock releases the lock and closes the file handle.
// Does not die on errors, at this point it wouldn't really do any good.
func ReleaseFileLock(filePath string) {
	lockFiles.mutex.Lock()
	defer lockFiles.mutex.Unlock()

	lockFile, ok := lockFiles.files[filePath]
	if !ok {
		log.Errorf("Lock file not acquired!")
		return
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
		log.Errorf("Failed to release lock: %s", err) // No point making this fatal really
	}
	if err := lockFile.Close(); err != nil {
		log.Errorf("Failed to close lock file: %s", err)
	}
	delete(lockFiles.files, filePath)
}

// ReadLastOperationOrDie reads the last operation performed from the lock file. Dies if unsuccessful.
func ReadLastOperationOrDie() []string {
	contents, err := ioutil.ReadFile(repoLockFilePath)
	if err != nil || len(contents) == 0 {
		log.Fatalf("Sorry OP, can't read previous operation :(")
	}
	return strings.Split(strings.TrimSpace(string(contents)), " ")
}

func ReleaseRepoLock() {
	ReleaseFileLock(repoLockFilePath)
}

func openLockFile(filePath string) *os.File {
	var lockFile *os.File
	var err error
	os.MkdirAll(path.Dir(filePath), DirPermissions)
	// TODO(pebers): This doesn't seem quite as intended, I think the file still gets truncated sometimes.
	//               Not sure why since I'm not passing O_TRUNC...
	if lockFile, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644); err != nil {
		log.Fatalf("Failed to acquire lock: %s", err)
	}

	return lockFile
}
