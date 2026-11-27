package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

type mode os.FileMode

func (m mode) IsDir() bool {
	return os.FileMode(m).IsDir()
}

func (m mode) IsRegular() bool {
	return os.FileMode(m).IsRegular()
}

func (m mode) IsSymlink() bool {
	return os.FileMode(m)&os.ModeSymlink != 0
}

func (m mode) ModeType() os.FileMode {
	return os.FileMode(m)
}

// Walk implements an equivalent to filepath.Walk.
func Walk(rootPath string, callback func(name string, isDir bool) error) error {
	return WalkMode(rootPath, func(name string, entry fs.DirEntry) error {
		return callback(name, entry.IsDir())
	})
}

// WalkMode is like Walk but the callback receives an additional type specifying the file mode type.
// N.B. This only includes the bits of the mode that determine the mode type, not the permissions.
func WalkMode(rootPath string, callback func(name string, entry fs.DirEntry) error) error {
	return filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return callback(path, entry)
	})
}
