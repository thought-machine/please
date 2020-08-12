package clean

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/fs"
)

func TestAsyncDeleteDir(t *testing.T) {
	err := os.MkdirAll("test_dir/a/b/c", os.ModeDir|0775)
	assert.NoError(t, err)
	err = AsyncDeleteDir("test_dir")
	assert.NoError(t, err)
	for i := 0; i < 100; i++ {
		if !fs.PathExists("test_dir") {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.False(t, fs.PathExists("test_dir"))
}
