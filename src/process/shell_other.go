//go:build !windows
// +build !windows

package process

// shellInitArgs stop bash reading the user's profile and rc files, so build actions don't
// pick up anything from the invoking user's environment.
var shellInitArgs = []string{"--noprofile", "--norc"}
