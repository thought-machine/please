//go:build !windows
// +build !windows

package fs

// isSymlinkPrivilegeError reports whether an error from os.Symlink means the OS refused for
// want of a privilege. Only Windows does that.
func isSymlinkPrivilegeError(error) bool { return false }
