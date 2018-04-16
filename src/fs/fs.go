// Package fs provides various filesystem helpers.
package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"syscall"
)

// DirPermissions are the default permission bits we apply to directories.
const DirPermissions = os.ModeDir | 0775

// EnsureDir ensures that the directory of the given file has been created.
func EnsureDir(filename string) error {
	return os.MkdirAll(path.Dir(filename), DirPermissions)
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

// IsSameFile returns true if two filenames describe the same underlying file (i.e. inode)
func IsSameFile(a, b string) bool {
	i1, err1 := getInode(a)
	i2, err2 := getInode(b)
	return err1 == nil && err2 == nil && i1 == i2
}

// getInode returns the inode of a file.
func getInode(filename string) (uint64, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	s, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("Not a syscall.Stat_t")
	}
	return s.Ino, nil
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
	if err := os.RemoveAll(to); err != nil {
		return err
	}
	dir, file := path.Split(to)
	if err := os.MkdirAll(dir, DirPermissions); err != nil {
		return err
	}
	tempFile, err := ioutil.TempFile(dir, file)
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
	return os.Rename(tempFile.Name(), to)
}

// CopyOrLinkFile either copies or hardlinks a file based on the link argument.
// Falls back to a copy if link fails and fallback is true.
func CopyOrLinkFile(from, to string, mode os.FileMode, link, fallback bool) error {
	if link {
		if err := os.Link(from, to); err == nil || !fallback {
			return err
		} else if runtime.GOOS == "darwin" && os.IsNotExist(err) {
			// There is an awkward issue on OSX where links to symlinks actually try to link
			// to the target rather than the link itself. In that case we try to recreate
			// a similar symlink at the destination to work around.
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
