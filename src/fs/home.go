package fs

import (
	"os"
	"regexp"
	"strings"
)

var home = os.Getenv("HOME")
var homeRex = regexp.MustCompile("(?:^|:)(~(?:[/:]|$))")

// ExpandHomePath expands all prefixes of ~ without a user specifier to $HOME.
func ExpandHomePath(path string) string {
	return ExpandHomePathTo(path, home)
}

// ExpandHomePathTo expands all prefixes of ~ without a user specifier to the given string.
func ExpandHomePathTo(path, to string) string {
	return homeRex.ReplaceAllStringFunc(path, func(subpath string) string {
		return strings.Replace(subpath, "~", to, -1)
	})
}
