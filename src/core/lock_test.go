package core

import (
	"os"
	"strconv"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenLockFile(t *testing.T) {
	file, err := openLockFile("path/to/file")

	assert.NoError(t, err)
	assert.IsType(t, &os.File{}, file)

	file.Close()
}

func TestOpenRepoLockFile(t *testing.T) {
	// Open repo lock file.
	err := openRepoLockFile()
	assert.NoError(t, err)
	assert.IsType(t, &os.File{}, repoLockFile)

	// Store current value.
	previousRepoLockFile := repoLockFile

	// Try to "open" repo lock file again.
	err = openRepoLockFile()
	assert.NoError(t, err)
	assert.Equal(t, previousRepoLockFile, repoLockFile)

	ReleaseRepoLock()
}

func TestAcquireSharedRepoRoot(t *testing.T) {
	AcquireSharedRepoLock()
	defer ReleaseRepoLock()

	assert.IsType(t, &os.File{}, repoLockFile)

	contents, err := os.ReadFile(repoLockFile.Name())
	assert.Equal(t, strconv.Itoa(os.Getpid()), string(contents))
	assert.NoError(t, err)
}

func TestAcquireExclusiveRepoRoot(t *testing.T) {
	AcquireSharedRepoLock()
	defer ReleaseRepoLock()

	assert.IsType(t, &os.File{}, repoLockFile)

	contents, err := os.ReadFile(repoLockFile.Name())
	assert.Equal(t, strconv.Itoa(os.Getpid()), string(contents))
	assert.NoError(t, err)
}

func TestAcquireRepoRootOverride(t *testing.T) {
	err := acquireRepoLock(syscall.LOCK_SH | syscall.LOCK_NB)
	assert.NoError(t, err)

	// It is able to immediately override the lock mode since it uses the same file descriptor.
	err = acquireRepoLock(syscall.LOCK_EX | syscall.LOCK_NB)
	assert.NoError(t, err)

	ReleaseRepoLock()
}

// This attempts to mimic how 2 plz processes acquire a shared repo lock.
func TestAcquireSharedRepoRootTwice(t *testing.T) {
	// 1st process.
	err := acquireRepoLock(syscall.LOCK_SH | syscall.LOCK_NB)
	assert.NoError(t, err)

	// Keep file descriptor reference alive.
	processOneRepoLockFile := repoLockFile
	defer ReleaseFileLock(processOneRepoLockFile)

	// 2nd process.
	repoLockFile = nil // Reset.
	// It is able to immediately acquire another shared lock via a different file descriptor.
	err = acquireRepoLock(syscall.LOCK_SH | syscall.LOCK_NB)
	assert.NoError(t, err)

	ReleaseRepoLock()
}

// This attempts to mimic how 1 plz process acquires a shared repo lock and another tries to acquire an exclusive one.
func TestAcquireSharedAndExclusiveRepoRoot(t *testing.T) {
	// 1st process.
	err := acquireRepoLock(syscall.LOCK_SH | syscall.LOCK_NB)
	assert.NoError(t, err)

	// Keep file descriptor reference alive.
	processOneRepoLockFile := repoLockFile
	defer ReleaseFileLock(processOneRepoLockFile)

	// 2nd process.
	repoLockFile = nil // Reset.
	// It errors immediately trying to acquire an exclusive lock as a shared one already exists from process 1.
	err = acquireRepoLock(syscall.LOCK_EX | syscall.LOCK_NB)
	assert.Error(t, err)

	ReleaseRepoLock()
}

// This attempts to mimic how 1 plz process acquires an exclusive repo lock and another tries to acquire a shared one.
func TestAcquireExclusiveAndSharedRepoRoot(t *testing.T) {
	// 1st process.
	err := acquireRepoLock(syscall.LOCK_EX | syscall.LOCK_NB)
	assert.NoError(t, err)

	// Keep file descriptor reference alive.
	processOneRepoLockFile := repoLockFile
	defer ReleaseFileLock(processOneRepoLockFile)

	// 2nd process.
	repoLockFile = nil // Reset.
	// It errors immediately trying to acquire a shared lock as an exclusive one already exists from process 1.
	err = acquireRepoLock(syscall.LOCK_SH | syscall.LOCK_NB)
	assert.Error(t, err)

	ReleaseRepoLock()
}

func TestReleaseRepoLock(t *testing.T) {
	AcquireSharedRepoLock()

	ReleaseRepoLock()
	assert.Nil(t, repoLockFile)
}

func TestAcquireExclusiveFileLock(t *testing.T) {
	file := AcquireExclusiveFileLock("path/to/file")
	defer ReleaseFileLock(file)

	assert.IsType(t, &os.File{}, file)

	contents, err := os.ReadFile("path/to/file")
	assert.Equal(t, strconv.Itoa(os.Getpid()), string(contents))
	assert.NoError(t, err)
}

// This attempts to mimic how 1 plz process acquires an exclusive file lock and another tries to do the same thing to the same file.
func TestAcquireExclusiveFileLockTwice(t *testing.T) {
	// 1st process.
	fd1, err := openAndAcquireLockFile("path/to/file", syscall.LOCK_EX|syscall.LOCK_NB)
	assert.NoError(t, err)

	// Keep file descriptor reference alive.
	processOneLockFile := fd1
	defer ReleaseFileLock(processOneLockFile)

	// 2nd process.
	// It errors immediately trying to acquire an exclusive lock as the same lock mode was already placed by process 1.
	fd2, err := openAndAcquireLockFile("path/to/file", syscall.LOCK_EX|syscall.LOCK_NB)
	assert.Error(t, err)

	ReleaseFileLock(fd2)
}

func TestReleaseFileLock(t *testing.T) {
	file := AcquireExclusiveFileLock("path/to/file")

	ReleaseFileLock(file)
	err := file.Close()
	assert.Error(t, err, "file already closed")
}
