package fs

// removeNeedsWritableFiles is whether a file has to be writable for its parent directory to be
// removable. Windows refuses to delete a file carrying FILE_ATTRIBUTE_READONLY - which is what
// os.Chmod manipulates there - and the read-only attribute on a directory means something else
// entirely, so the files themselves have to be cleared.
const removeNeedsWritableFiles = true
