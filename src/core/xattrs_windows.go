package core

// defaultXattrs is whether we try to record file metadata in extended attributes by default.
// Windows has no equivalent, so we always fall back to writing separate files.
const defaultXattrs = false
