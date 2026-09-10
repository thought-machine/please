//go:build !windows
// +build !windows

package fs

// ExplainUnrunnable returns extra context for a file that could not be executed, or an empty
// string if there is nothing useful to add. There never is on Unix, where the executable bit
// decides and the error already says so.
func ExplainUnrunnable(string) string { return "" }
