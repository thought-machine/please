//go:build !windows
// +build !windows

package run

// envNamesEqual reports whether two environment variable names refer to the same variable.
// Unix environment names are case-sensitive, so this is plain equality.
func envNamesEqual(a, b string) bool { return a == b }
