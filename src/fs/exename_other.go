//go:build !windows
// +build !windows

package fs

// PathSeparators are the characters that separate elements of a path.
const PathSeparators = "/"

// ExeSuffix is what an executable's filename ends in. Unix decides by the executable bit
// rather than the name, so there is nothing to add.
const ExeSuffix = ""

// ExecutableNames returns the filenames to try when searching the path for an executable
// called name. On Unix an executable is just a file with the executable bit set, so there is
// only ever one candidate.
func ExecutableNames(name string) []string {
	return []string{name}
}
