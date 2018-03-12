// +build !bootstrap

package output

import (
	"fmt"
	"os"

	"cli"
)

// printf is used throughout this package to print something to stderr with some niceties
// around ANSI formatting codes.
func printf(format string, args ...interface{}) {
	if !cli.StdErrIsATerminal {
		format = cli.StripAnsi.ReplaceAllString(format, "")
	}
	fmt.Fprintf(os.Stderr, format, args...)
}
