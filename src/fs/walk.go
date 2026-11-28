package fs

import (
	"io/fs"
	"path/filepath"
)

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
