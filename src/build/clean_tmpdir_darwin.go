package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/thought-machine/please/src/fs"
)

// cleanTmpDir removes a target's temporary build directory.
//
// On macOS, the operating system automatically creates a ~/Library directory
// structure for every HOME directory it observes in use by a process. This is
// documented in Apple's File System Programming Guide:
//
//	https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/MacOSXDirectories/MacOSXDirectories.html
//
// Several per-user system daemons (cfprefsd, lsd, etc.) monitor process HOME
// values and lazily create $HOME/Library/ and its subdirectories (Preferences,
// Caches, etc.) for any new HOME path they observe. This creation is
// asynchronous and can occur after the process that triggered it has already
// exited.
//
// Since the build environment sets HOME=tmpDir (to isolate each target's build
// from the user's home directory), these daemons may create a Library/
// directory inside the target's tmpDir during or shortly after the build
// completes. When os.RemoveAll deletes the tmpDir contents and then attempts
// to remove the now-empty directory, a daemon may have re-created Library/ in
// the interim, causing the removal to fail with ENOTEMPTY.
//
// We handle this by detecting the Library/ directory after an ENOTEMPTY
// failure, removing it, and retrying with bounded backoff to allow the daemons
// to settle.
func cleanTmpDir(tmpDir string) error {
	err := fs.RemoveAll(tmpDir)
	if err == nil || !errors.Is(err, syscall.ENOTEMPTY) {
		return err
	}

	libDir := filepath.Join(tmpDir, "Library")
	const maxAttempts = 3
	for attempt := range maxAttempts {
		if attempt > 0 {
			time.Sleep(time.Second)
		}
		if info, serr := os.Stat(libDir); serr != nil || !info.IsDir() {
			// Library/ is not the problem; return the original error.
			return err
		}
		os.RemoveAll(libDir)
		if rerr := os.Remove(tmpDir); rerr == nil {
			return nil
		} else if !errors.Is(rerr, syscall.ENOTEMPTY) {
			return rerr
		}
	}
	return fmt.Errorf("failed to remove %s after %d attempts to clean macOS Library dir: %w", tmpDir, maxAttempts, err)
}
