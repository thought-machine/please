//go:build !windows
// +build !windows

package fs

// ExecutableNames returns the filenames to try when searching the path for an executable
// called name. On Unix an executable is just a file with the executable bit set, so there is
// only ever one candidate.
func ExecutableNames(name string) []string {
	return []string{name}
}
