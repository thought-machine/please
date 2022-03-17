package fs

import (
	"os"
	"path"
)

// CopyOrLinkFile either copies or hardlinks a file based on the link argument.
// Falls back to a copy if link fails and fallback is true.
func CopyOrLinkFile(from, to string, fromMode, toMode os.FileMode, link, fallback bool) error {
	if link {
		if (fromMode & os.ModeSymlink) != 0 {
			// Don't try to hard-link to a symlink, that doesn't work reliably across all platforms.
			// Instead recreate an equivalent symlink in the new location.
			dest, err := os.Readlink(from)
			if err != nil {
				return err
			}
			return os.Symlink(dest, to)
		}
		if err := os.Link(from, to); err == nil || !fallback {
			return err
		}

		// Linking would ignore toMode, using the same mode as the from file. We should make the fallback work the same
		// here.
		info, err := os.Lstat(from)
		if err != nil {
			return err
		}
		toMode = info.Mode()
	}
	return CopyFile(from, to, toMode)
}

// RecursiveCopy copies either a single file or a directory.
// 'mode' is the mode of the destination file.
func RecursiveCopy(from string, to string, mode os.FileMode) error {
	return RecursiveCopyOrLinkFile(from, to, mode, false, false)
}

// RecursiveLink hardlinks either a single file or a directory.
// Note that you can't hardlink directories so the behaviour is much the same as a recursive copy.
// If it can't link then it falls back to a copy.
// 'mode' is the mode of the destination file.
func RecursiveLink(from string, to string) error {
	return RecursiveCopyOrLinkFile(from, to, 0, true, true)
}

// RecursiveCopyOrLinkFile recursively copies or links a file or directory.
// 'mode' is the mode of the destination file.
// If 'link' is true then we'll hardlink files instead of copying them.
// If 'fallback' is true then we'll fall back to a copy if linking fails.
func RecursiveCopyOrLinkFile(from string, to string, mode os.FileMode, link, fallback bool) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return WalkMode(from, func(name string, fileMode Mode) error {
			dest := path.Join(to, name[len(from):])
			if fileMode.IsDir() {
				return os.MkdirAll(dest, DirPermissions)
			}
			return CopyOrLinkFile(name, dest, fileMode.ModeType(), mode, link, fallback)
		})
	}
	return CopyOrLinkFile(from, to, info.Mode(), mode, link, fallback)
}

type LinkFunc func(string, string) error

// LinkIfNotExists creates dest as a link to src if it doesn't already exist.
func LinkIfNotExists(src, dest string, f LinkFunc) {
	if PathExists(dest) {
		return
	}
	Walk(src, func(name string, isDir bool) error {
		if !isDir {
			fullDest := path.Join(dest, name[len(src):])
			if err := EnsureDir(fullDest); err != nil {
				log.Warning("Failed to create directory for %s: %s", fullDest, err)
			} else if err := f(name, fullDest); err != nil && !os.IsExist(err) {
				log.Warning("Failed to create %s: %s", fullDest, err)
			}
		}
		return nil
	})
}

func LinkDestination(src, dest string, f LinkFunc) {
	Walk(src, func(name string, isDir bool) error {
		if !isDir {
			fullDest := path.Join(dest, name[len(src):])
			if err := EnsureDir(fullDest); err != nil {
				log.Warning("Failed to create directory for %s: %s", fullDest, err)
			} else if err := f(name, fullDest); err != nil && !os.IsExist(err) {
				log.Warning("Failed to create %s: %s", fullDest, err)
			}
		}
		return nil
	})
}
