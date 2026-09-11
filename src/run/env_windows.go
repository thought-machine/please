package run

import "strings"

// envNamesEqual reports whether two environment variable names refer to the same variable.
//
// Windows environment names are case-insensitive, and the OS keeps its own spelling: setting
// PATH updates the variable it already has, which it stores as Path. Comparing names exactly
// therefore fails to find it, and a caller that meant to replace an entry appends a second one
// instead. os/exec happens to paper over that by deduplicating case-insensitively itself, but
// nothing else does - ExecReplace and the audit log both see the duplicate.
func envNamesEqual(a, b string) bool { return strings.EqualFold(a, b) }
