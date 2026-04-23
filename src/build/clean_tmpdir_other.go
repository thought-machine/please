//go:build !darwin
// +build !darwin

package build

import "github.com/thought-machine/please/src/fs"

// cleanTmpDir removes a target's temporary build directory.
func cleanTmpDir(tmpDir string) error {
	return fs.RemoveAll(tmpDir)
}
