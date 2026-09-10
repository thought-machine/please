package fs

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isSymlinkPrivilegeError reports whether an error from os.Symlink means the OS refused for
// want of a privilege. Creating a symlink on Windows needs either Developer Mode or
// SeCreateSymbolicLinkPrivilege, neither of which an ordinary user has by default.
func isSymlinkPrivilegeError(err error) bool {
	return errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD)
}
