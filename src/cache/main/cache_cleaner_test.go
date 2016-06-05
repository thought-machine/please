package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheEntriesSortByTime(t *testing.T) {
	entries := CacheEntries{
		CacheEntry{Path: "path1", Size: 1000, Atime: 1449488976},
		CacheEntry{Path: "path2", Size: 1000, Atime: 1449688978},
		CacheEntry{Path: "path3", Size: 1000, Atime: 1449588977},
	}
	sort.Sort(entries)
	// Oldest first.
	assert.Equal(t, "path1", entries[0].Path)
	assert.Equal(t, "path3", entries[1].Path)
	assert.Equal(t, "path2", entries[2].Path)
}

func TestCacheEntriesSortBySizeAfterTime(t *testing.T) {
	entries := CacheEntries{
		CacheEntry{Path: "path1", Size: 10, Atime: 1449488976},
		CacheEntry{Path: "path2", Size: 100000, Atime: 1449488976},
		CacheEntry{Path: "path3", Size: 1000, Atime: 1449488976},
	}
	sort.Sort(entries)
	// Largest first.
	assert.Equal(t, "path2", entries[0].Path)
	assert.Equal(t, "path3", entries[1].Path)
	assert.Equal(t, "path1", entries[2].Path)
}

func TestCacheEntriesSortingWithTolerance(t *testing.T) {
	entries := CacheEntries{
		CacheEntry{Path: "path1", Size: 10, Atime: 1449488976},
		CacheEntry{Path: "path2", Size: 100000, Atime: 1449488978},
		CacheEntry{Path: "path3", Size: 1000, Atime: 1449488977},
	}
	sort.Sort(entries)
	// Similar to the previous test, but times aren't exactly the same; we should prefer to
	// delete the 100kB file over the 10B one, it being two seconds newer shouldn't be enough
	// to save it.
	assert.Equal(t, "path2", entries[0].Path)
	assert.Equal(t, "path3", entries[1].Path)
	assert.Equal(t, "path1", entries[2].Path)
}
