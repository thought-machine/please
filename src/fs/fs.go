// Package fs provides various filesystem helpers.
package fs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/cli/logging"
)

var log = logging.Log

// DirPermissions are the default permission bits we apply to directories.
const DirPermissions = os.ModeDir | 0775

// EnsureDir ensures that the directory of the given file has been created.
func EnsureDir(filename string) error {
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, DirPermissions)
	if err != nil && FileExists(dir) {
		// It looks like this is a file and not a directory. Attempt to remove it; this can
		// happen in some cases if you change a rule from outputting a file to a directory.
		log.Warning("Attempting to remove file %s; a subdirectory is required", dir)
		if err2 := os.Remove(dir); err2 == nil {
			err = os.MkdirAll(dir, DirPermissions)
		} else {
			log.Error("%s", err2)
		}
	}
	return err
}

// OpenDirFile ensures that the directory of the given file has been created before
// calling the underlying os.OpenFile function.
func OpenDirFile(filename string, flag int, perm os.FileMode) (*os.File, error) {
	if err := EnsureDir(filename); err != nil {
		return nil, err
	}
	return os.OpenFile(filename, flag, perm)
}

// PathExists returns true if the given path exists, as a file or a directory.
func PathExists(filename string) bool {
	_, err := os.Lstat(filename)
	return err == nil
}

// FileExists returns true if the given path exists and is a file.
func FileExists(filename string) bool {
	info, err := os.Lstat(filename)
	return err == nil && !info.IsDir()
}

// IsSymlink returns true if the given path exists and is a symlink.
func IsSymlink(filename string) bool {
	info, err := os.Lstat(filename)
	return err == nil && (info.Mode()&os.ModeSymlink) != 0
}

// IsSameFile returns true if two filenames describe the same underlying file
// (i.e. inode for Unix and potentially file path names for other OS's)
func IsSameFile(a, b string) bool {
	i1, err1 := getFileInfo(a)
	i2, err2 := getFileInfo(b)
	return err1 == nil && err2 == nil && os.SameFile(i1, i2)
}

// getFileInfo returns the FileInfo of a file.
func getFileInfo(filename string) (os.FileInfo, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

// CopyFile copies a file from 'from' to 'to', with an attempt to perform a copy & rename
// to avoid chaos if anything goes wrong partway.
func CopyFile(from string, to string, mode os.FileMode) error {
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fromFile.Close()
	return WriteFile(fromFile, to, mode)
}

// WriteFile writes data from a reader to the file named 'to', with an attempt to perform
// a copy & rename to avoid chaos if anything goes wrong partway.
func WriteFile(fromFile io.Reader, to string, mode os.FileMode) error {
	dir, file := filepath.Split(to)
	if dir != "" {
		if err := os.MkdirAll(dir, DirPermissions); err != nil {
			return err
		}
	}
	tempFile, err := os.CreateTemp(dir, file)
	if err != nil {
		return err
	}
	if _, err := io.Copy(tempFile, fromFile); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	// OK, now file is written; adjust permissions appropriately.
	if mode == 0 {
		mode = 0664
	}
	if err := os.Chmod(tempFile.Name(), mode); err != nil {
		return err
	}
	// And move it to its final destination.
	return renameFile(tempFile.Name(), to)
}

// IsDirectory checks if a given path is a directory
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsPackage returns true if the given directory name is a package (i.e. contains a build file)
func IsPackage(buildFileNames []string, name string) bool {
	for _, buildFileName := range buildFileNames {
		if FileExists(filepath.Join(name, buildFileName)) {
			return true
		}
	}
	return false
}

// Try to gracefully rename the file as the os.Rename does not work across
// filesystems and on most Linux systems /tmp is mounted as tmpfs
func renameFile(from, to string) (err error) {
	err = os.Rename(from, to)
	if err == nil {
		return nil
	}
	err = copyFile(from, to)
	if err != nil {
		return err
	}
	err = RemoveAll(from)
	if err != nil {
		return err
	}
	return nil
}

func copyFile(from, to string) (err error) {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	si, err := os.Stat(from)
	if err != nil {
		return err
	}
	err = os.Chmod(to, si.Mode())
	if err != nil {
		return err
	}

	return nil
}

// RemoveAll will try and remove the path with `os.RemoveAll`; if that fails with a permission error,
// it will attempt to adjust permissions to make things writable, then remove them.
func RemoveAll(path string) error {
	if err := os.RemoveAll(path); err == nil || err != os.ErrPermission {
		return nil
	} else if err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if d.IsDir() {

		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to remove directory %s (could not make writable: %w", path, err)
	}
	return os.RemoveAll(path)
}
