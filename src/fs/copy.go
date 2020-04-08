package fs

import (
	"os"
	"path"
)

// CopyOrLinkFile either copies or hardlinks a file based on the link argument.
// Falls back to a copy if link fails and fallback is true.
func copyOrLinkFile(from, to string, fromMode, toMode os.FileMode, link, fallback bool) error {
	if link {
		if (fromMode&os.ModeSymlink) != 0 {
			// Don't try to hard-link to a symlink, that doesn't work reliably across all platforms.
			// Instead recreate an equivalent symlink in the new location.
			dest, err := os.Readlink(from)
			if err != nil {
				return err
			}
			return os.Symlink(dest, to)
		}
		return os.Link(from, to)
	}
	return CopyFile(from, to, toMode)
}

// RecursiveCopy copies either a single file or a directory.
// 'mode' is the mode of the destination file.
func RecursiveCopy(from string, to string, mode os.FileMode) error {
	return recursiveCopyOrLinkFile(from, to, mode, false, false)
}

// RecursiveLink hardlinks either a single file or a directory.
// Note that you can't hardlink directories so the behaviour is much the same as a recursive copy.
// If it can't link then it falls back to a copy.
// 'mode' is the mode of the destination file.
func RecursiveLink(from string, to string, mode os.FileMode) error {
	return recursiveCopyOrLinkFile(from, to, mode, true, true)
}

// recursiveCopyOrLinkFile recursively copies or links a file or directory.
// 'mode' is the mode of the destination file.
// If 'link' is true then we'll hardlink files instead of copying them.
// If 'fallback' is true then we'll fall back to a copy if linking fails.
func recursiveCopyOrLinkFile(from string, to string, mode os.FileMode, link, fallback bool) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return WalkMode(from, func(name string, isDir bool, fileMode os.FileMode) error {
			dest := path.Join(to, name[len(from):])
			if isDir {
				return os.MkdirAll(dest, DirPermissions)
			}
			return copyOrLinkFile(name, dest, fileMode, mode, link, fallback)
		})
	}
	return copyOrLinkFile(from, to, info.Mode(), mode, link, fallback)
}
