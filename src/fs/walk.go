package fs

import (
	"os"

	"github.com/karrick/godirwalk"
)

// Walk implements an equivalent to filepath.Walk.
// It's implemented over github.com/karrick/godirwalk but the provided interface doesn't use that
// to make it a little easier to handle.
func Walk(rootPath string, callback func(name string, isDir bool) error) error {
	return WalkMode(rootPath, func(name string, isDir bool, mode os.FileMode) error {
		return callback(name, isDir)
	})
}

// WalkMode is like Walk but the callback receives an additional type specifying the file mode type.
// N.B. This only includes the bits of the mode that determine the mode type, not the permissions.
func WalkMode(rootPath string, callback func(name string, isDir bool, mode os.FileMode) error) error {
	// Compatibility with filepath.Walk which allows passing a file as the root argument.
	if info, err := os.Lstat(rootPath); err != nil {
		return err
	} else if !info.IsDir() {
		return callback(rootPath, false, info.Mode())
	}
	return godirwalk.Walk(rootPath, &godirwalk.Options{Callback: func(name string, info *godirwalk.Dirent) error {
		return callback(name, info.IsDir(), info.ModeType())
	}})
}
