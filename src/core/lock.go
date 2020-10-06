// Contains utility functions for managing an exclusive lock file.
// Based on flock() underneath so

package core

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/xattr"
)

const lockFilePath = "plz-out/.lock"

var lockFile *os.File

// AcquireRepoLock opens the lock file and acquires the lock.
// Dies if the lock cannot be successfully acquired.
func AcquireRepoLock(state *BuildState) {
	var err error
	// There is of course technically a bit of a race condition between the file & flock operations here,
	// but it shouldn't matter much since we're trying to mutually exclude plz processes started by the user
	// which (one hopes) they wouldn't normally do simultaneously.
	os.MkdirAll(path.Dir(lockFilePath), DirPermissions)
	// TODO(pebers): This doesn't seem quite as intended, I think the file still gets truncated sometimes.
	//               Not sure why since I'm not passing O_TRUNC...
	if lockFile, err = os.OpenFile(lockFilePath, os.O_RDWR|os.O_CREATE, 0644); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to acquire lock: %s", err)
	} else if lockFile, err = os.Create(lockFilePath); err != nil {
		log.Fatalf("Failed to create lock: %s", err)
	}
	acquireLockfile(lockFile)

	// Record the operation performed.
	if _, err = lockFile.Seek(0, io.SeekStart); err == nil {
		if n, err := lockFile.Write([]byte(strings.Join(os.Args[1:], " ") + "\n")); err == nil {
			lockFile.Truncate(int64(n))
		}
	}

	// Quick test of xattrs; we don't keep trying to use them if they fail here.
	if state != nil && state.XattrsSupported {
		if err := xattr.Set(lockFilePath, "user.plz_build", []byte("lock")); err != nil {
			log.Warning("xattrs are not supported on this filesystem, using fallbacks")
			state.DisableXattrs()
		}
	}
}

// ReleaseRepoLock releases the lock and closes the file handle.
// Does not die on errors, at this point it wouldn't really do any good.
func ReleaseRepoLock() {
	if lockFile == nil {
		log.Errorf("Lock file not acquired!")
		return
	}
	if err := releaseLockFile(lockFile); err != nil {
		log.Errorf("Failed to release lock: %s", err) // No point making this fatal really
	}
	if err := lockFile.Close(); err != nil {
		log.Errorf("Failed to close lock file: %s", err)
	}
}

// ReadLastOperationOrDie reads the last operation performed from the lock file. Dies if unsuccessful.
func ReadLastOperationOrDie() []string {
	contents, err := ioutil.ReadFile(lockFilePath)
	if err != nil || len(contents) == 0 {
		log.Fatalf("Sorry OP, can't read previous operation :(")
	}
	return strings.Split(strings.TrimSpace(string(contents)), " ")
}
