//go:build !windows
// +build !windows

package core

// normalisePathSeparators is a no-op where the path separator is already a forward slash.
func (env BuildEnv) normalisePathSeparators() {}
