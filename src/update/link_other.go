//go:build !windows
// +build !windows

package update

import (
	"os"

	"github.com/thought-machine/please/src/fs"
)

// linkFile points globalFile at downloadedFile, replacing whatever was there before.
func linkFile(downloadedFile, globalFile string) error {
	if err := fs.RemoveAll(globalFile); err != nil {
		return err
	}
	return os.Symlink(downloadedFile, globalFile)
}

// cleanStaleFiles does nothing here; only Windows can fail to replace a file and have to
// leave the old one behind.
func cleanStaleFiles(string) {}
