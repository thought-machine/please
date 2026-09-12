package fs

import (
	"golang.org/x/sys/windows"
)

// IsWine reports whether this process is running under Wine rather than on Windows.
//
// It exists so that a test can skip where Wine is known to lie, rather than skipping on Windows
// wholesale and telling us nothing about the platform we actually care about. Wine's symlinks
// are the case that forced it: os.Symlink reports success and produces a link os.Lstat cannot
// find, so a test written against real behaviour fails there for a reason that is not a bug.
//
// Detected by a function only Wine exports. Wine documents this as the supported way to tell.
func IsWine() bool {
	return windows.NewLazySystemDLL("ntdll.dll").NewProc("wine_get_version").Find() == nil
}
