//go:build !windows
// +build !windows

package fs

// IsWine reports whether this process is running under Wine rather than on Windows. Nothing
// that is not a Windows binary is.
func IsWine() bool { return false }
