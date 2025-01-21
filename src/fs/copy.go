package fs

import (
	"fmt"
	"os"
	"path/filepath"
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
			dest := filepath.Join(to, name[len(from):])
			if fileMode.IsDir() {
				return os.MkdirAll(dest, DirPermissions)
			}
			if fileMode.IsSymlink() {
				return copySymlink(name, dest)
			}
			return CopyOrLinkFile(name, dest, fileMode.ModeType(), mode, link, fallback)
		})
	}
	return CopyOrLinkFile(from, to, info.Mode(), mode, link, fallback)
}

// copySymlink will resolve the symlink and create an equivalent symlink at dest. Assumes the symlink is relative, not absolute.
func copySymlink(name, dest string) error {
	resolvedPath, err := os.Readlink(name)
	if err != nil {
		return err
	}

	if resolvedPath == "../mocha/bin/_mocha" {
		log.Warningf("linking %v to %v", resolvedPath, dest)
	}

	return os.Symlink(resolvedPath, dest)
}

type LinkFunc func(string, string) error

// LinkIfNotExists creates dest as a link to src if it doesn't already exist.
func LinkIfNotExists(src, dest string, f LinkFunc) {
	if PathExists(dest) {
		return
	}
	Walk(src, func(name string, isDir bool) error {
		if !isDir {
			fullDest := filepath.Join(dest, name[len(src):])
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
			fullDest := filepath.Join(dest, name[len(src):])
			if err := EnsureDir(fullDest); err != nil {
				log.Warning("Failed to create directory for %s: %s", fullDest, err)
			} else if err := f(name, fullDest); err != nil && !os.IsExist(err) {
				log.Warning("Failed to create %s: %s", fullDest, err)
			}
		}
		return nil
	})
}

// Link creates dest as a hard link to the src, replacing existing dest
// links to support cases where hard link metadata is not stored (e.g. with
// `git`).
func Link(src, dest string) error {
	if PathExists(dest) {
		// remove existing hard links as git won't follow them
		if err := os.Remove(dest); err != nil {
			return fmt.Errorf("could not remove link %s: %w", dest, err)
		}
	}

	return os.Link(src, dest)
}

// Symlink creates dest as symbolic link to the src, skipping if symbolic link
// already exists.
func Symlink(src, dest string) error {
	if !PathExists(src) {
		return fmt.Errorf("%s: %w", src, os.ErrNotExist)
	}

	if PathExists(dest) {
		fileInfo, err := os.Lstat(dest)
		if err != nil {
			return fmt.Errorf("could get Lstat %s: %w", dest, err)
		}
		if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			// is already a symbolic link
			return nil
		}

		// remove existing files that aren't symbolic links
		if err := os.Remove(dest); err != nil {
			return fmt.Errorf("could not remove link %s: %w", dest, err)
		}
	}

	return os.Symlink(src, dest)
}
