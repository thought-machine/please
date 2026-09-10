//go:build !windows
// +build !windows

package core

// sandboxSupported reports whether this platform can isolate a build action at all.
// Everywhere but Windows there is at least a sandbox tool to hand the action to, even if what
// it does varies; on Linux it does the whole job.
func sandboxSupported() bool { return true }
