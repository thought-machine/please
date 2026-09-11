package fs

import (
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

// removeNeedsWritableFiles is whether a file has to be writable for its parent directory to be
// removable. Windows refuses to delete a file carrying FILE_ATTRIBUTE_READONLY - which is what
// os.Chmod manipulates there - and the read-only attribute on a directory means something else
// entirely, so the files themselves have to be cleared.
const removeNeedsWritableFiles = true

// removeRetries is how many times to retry a removal that failed because something else had the
// file open, and how long to wait between attempts.
//
// Windows will not unlink or rename a file another handle has open, and real-time virus scanning
// opens files Please has just written, for as long as it takes to scan them. That makes this a
// transient failure rather than a permanent one, unlike every other error here. A handle that is
// genuinely held - by this process, or by something the user is running - outlives the retries
// and still fails, which is what we want.
const removeRetries = 10
const removeRetryDelay = 100 * time.Millisecond

// isTransientRemoveError reports whether a removal failed for a reason that may not still be
// true in a moment.
func isTransientRemoveError(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION)
}
