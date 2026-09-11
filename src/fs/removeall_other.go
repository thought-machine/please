//go:build !windows
// +build !windows

package fs

import "time"

// removeNeedsWritableFiles is whether a file has to be writable for its parent directory to be
// removable. On Unix only the directory's own permissions matter.
const removeNeedsWritableFiles = false

// removeRetries and removeRetryDelay are the Windows retry loop's settings; there is nothing to
// retry here, because Unix is happy to unlink a file that is still open.
const removeRetries = 1
const removeRetryDelay = time.Duration(0)

// isTransientRemoveError reports whether a removal failed for a reason that may not still be
// true in a moment. Nothing on Unix qualifies.
func isTransientRemoveError(error) bool { return false }
