package fs

import (
	"os"
	"regexp"
	"strings"

	"github.com/peterebden/go-deferred-regex"
)

var homeRex = deferredregex.DeferredRegex{Re: homePathRegex()}

// homePathRegex returns the pattern matching a bare ~ at the start of a path, or at the start
// of an entry within a PATH-style list. Both the list separator and the path separators are
// platform-specific, and Windows accepts either slash.
func homePathRegex() string {
	listSep := regexp.QuoteMeta(string(os.PathListSeparator))
	pathSeps := "/"
	if os.PathSeparator == '\\' {
		pathSeps = `/\\`
	}
	return `(?:^|` + listSep + `)(~(?:[` + pathSeps + listSep + `]|$))`
}

// ExpandHomePath expands all prefixes of ~ without a user specifier to the user's home directory.
func ExpandHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Same as the old behaviour of reading $HOME directly: if we can't tell, expand to nothing.
		home = ""
	}
	return ExpandHomePathTo(path, home)
}

// ExpandHomePathTo expands all prefixes of ~ without a user specifier to the given string.
func ExpandHomePathTo(path, to string) string {
	return homeRex.ReplaceAllStringFunc(path, func(subpath string) string {
		return strings.ReplaceAll(subpath, "~", to)
	})
}
