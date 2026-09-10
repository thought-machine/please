package watch

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatchKeyMatchesNativeSeparators(t *testing.T) {
	// The two sides of the comparison come from different places - our own slash-separated
	// source paths, and whatever fsnotify reports, which on Windows uses backslashes - so
	// they have to normalise to the same thing whichever separator each arrived with.
	assert.Equal(t, watchKey("src/core/foo.go"), watchKey(filepath.Join("src", "core", "foo.go")))
}

func TestWatchKeyIsIdempotent(t *testing.T) {
	once := watchKey(filepath.Join("src", "core", "foo.go"))
	assert.Equal(t, once, watchKey(once))
}
