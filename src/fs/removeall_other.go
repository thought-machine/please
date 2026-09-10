//go:build !windows
// +build !windows

package fs

// removeNeedsWritableFiles is whether a file has to be writable for its parent directory to be
// removable. On Unix only the directory's own permissions matter.
const removeNeedsWritableFiles = false
