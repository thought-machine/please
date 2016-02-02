// Tests for the core cache functionality
package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilesToClean(t *testing.T) {
	cachedFiles["test/artifact/1"] = &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -2),
		readCount:    0,
		size:         1000,
	}
	cachedFiles["test/artifact/2"] = &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -5),
		readCount:    0,
		size:         1000,
	}
	cachedFiles["test/artifact/3"] = &cachedFile{
		lastReadTime: time.Now().AddDate(0, 0, -1),
		readCount:    0,
		size:         1000,
	}
	totalSize = 3000

	paths := filesToClean(1700)
	assert.Equal(t, 2, len(paths))
}
