//go:build !windows
// +build !windows

package process

// DefaultShell is the shell we run build actions in if nothing else is configured.
const DefaultShell = "bash"

// DefaultShellArgs stop bash reading the user's profile and rc files, so build actions don't
// pick up anything from the invoking user's environment.
var DefaultShellArgs = []string{"--noprofile", "--norc"}
