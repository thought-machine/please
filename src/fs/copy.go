package fs

import (
	"os"
	"path"
	"runtime"
)

// CopyOrLinkFile either copies or hardlinks a file based on the link argument.
// Falls back to a copy if link fails and fallback is true.
func CopyOrLinkFile(from, to string, mode os.FileMode, link, fallback bool) error {
	if link {
		if err := os.Link(from, to); err == nil || !fallback {
			return err
		} else if runtime.GOOS != "linux" && os.IsNotExist(err) {
			// There is an awkward issue on several non-Linux platforms where links to
			// symlinks actually try to link to the target rather than the link itself.
			// In that case we try to recreate a similar symlink at the destination.
			if info, err := os.Lstat(from); err == nil && (info.Mode()&os.ModeSymlink) != 0 {
				dest, err := os.Readlink(from)
				if err != nil {
					return err
				}
				return os.Symlink(dest, to)
			}
			return err
		}
	}
	return CopyFile(from, to, mode)
}

// RecursiveCopy copies either a single file or a directory.
func RecursiveCopy(from string, to string, mode os.FileMode) error {
	return recursiveCopyOrLinkFile(from, to, mode, false, false)
}

// RecursiveLink hardlinks either a single file or a directory.
// Note that you can't hardlink directories so the behaviour is much the same as a recursive copy.
// If it can't link then it falls back to a copy.
func RecursiveLink(from string, to string, mode os.FileMode) error {
	return recursiveCopyOrLinkFile(from, to, mode, true, true)
}

// recursiveCopyOrLinkFile recursively copies or links a file or directory.
// If 'link' is true then we'll hardlink files instead of copying them.
// If 'fallback' is true then we'll fall back to a copy if linking fails.
func recursiveCopyOrLinkFile(from string, to string, mode os.FileMode, link, fallback bool) error {
	if info, err := os.Stat(from); err == nil && info.IsDir() {
		return WalkMode(from, func(name string, isDir bool, fileMode os.FileMode) error {
			dest := path.Join(to, name[len(from):])
			if isDir {
				return os.MkdirAll(dest, DirPermissions)
			}
			return CopyOrLinkFile(name, dest, mode, link, fallback)
		})
	}
	return CopyOrLinkFile(from, to, mode, link, fallback)
}
