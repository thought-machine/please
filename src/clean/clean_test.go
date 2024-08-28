package clean

import (
	"os"
	"slices"
	"strings"
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
	assert.Eventually(t, func() bool {
		return !dirExists(t, "test_dir")
	}, 10*time.Second, 100*time.Millisecond)
}

func dirExists(t *testing.T, name string) bool {
	if fs.PathExists(name) {
		return true
	}
	// Check it isn't still there as a .plz_clean dir
	entries, err := os.ReadDir(".")
	assert.NoError(t, err)
	return slices.ContainsFunc(entries, func(entry os.DirEntry) bool {
		return strings.Contains(entry.Name(), ".plz_clean")
	})
}

func TestMain(m *testing.M) {
	// This mimics what 'plz clean --rm x' does.
	if slices.Contains(os.Args, "--rm") {
		fs.RemoveAll(os.Args[len(os.Args)-1])
		return
	}
	os.Exit(m.Run())
}
