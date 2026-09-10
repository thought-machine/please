package update

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/fs"
)

// staleSuffix marks a file that was still in use when we tried to replace it.
const staleSuffix = ".stale"

// linkFile points globalFile at downloadedFile, replacing whatever was there before.
//
// Windows makes this harder than it is elsewhere, in two ways. Symlinks need Developer Mode or
// SeCreateSymbolicLinkPrivilege, which an ordinary user does not have, so we hard-link
// instead; that behaves the same for our purposes and needs no privilege on NTFS. And a
// running executable can be neither deleted nor written over, which matters because the file
// we are most often replacing is the Please that is doing the replacing. Windows does allow
// it to be renamed, so we move it aside and let a later run clear up.
func linkFile(downloadedFile, globalFile string) error {
	if err := removeOrRenameAside(globalFile); err != nil {
		return err
	}
	// Hard links fail across volumes and on filesystems that don't have them, so fall back to
	// a copy; it costs disk space but is always available.
	return fs.CopyOrLinkFile(downloadedFile, globalFile, 0555, 0555, true, true)
}

// removeOrRenameAside deletes a file, or renames it out of the way if it is in use.
func removeOrRenameAside(path string) error {
	if !fs.PathExists(path) {
		return nil
	}
	if err := fs.RemoveAll(path); err == nil {
		return nil
	}
	stale := path + staleSuffix
	// A previous update may have left one of these; it's fine if that one is still held too.
	if err := fs.RemoveAll(stale); err != nil {
		log.Debug("Couldn't remove %s: %s", stale, err)
	}
	log.Debug("Can't remove %s, renaming it to %s", path, stale)
	return os.Rename(path, stale)
}

// cleanStaleFiles removes anything an earlier update had to rename aside because it was in
// use at the time. Failures are expected and ignored; it may still be in use now.
func cleanStaleFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), staleSuffix) {
			if err := fs.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
				log.Debug("Couldn't remove stale file %s: %s", entry.Name(), err)
			}
		}
	}
}
