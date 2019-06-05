package core

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcquireRepoLock(t *testing.T) {
	// Grab the lock
	AcquireRepoLock(nil)
	// Now we should be able to open the file (ie. it exists)
	lockFile, err := os.Open(lockFilePath)
	assert.NoError(t, err)
	defer lockFile.Close()
	assert.Error(t, syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB))
	// Let it go again
	ReleaseRepoLock()
	// Now we can get it
	assert.NoError(t, syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB))
	// Let it go so the following tests aren't confused :)
	assert.NoError(t, syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN))
}

func TestReadLastOperation(t *testing.T) {
	assert.NoError(t, ioutil.WriteFile(lockFilePath, []byte("op plz"), 0644))
	assert.Equal(t, []string{"op", "plz"}, ReadLastOperationOrDie())
	// Can't really test a failure case because of the "or die" bit :(
}
