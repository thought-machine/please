// Package main implements an extremely cut-down version of jarcat
// which is used to break a circular dependency; jarcat depends on
// several third-party libraries, and we use the jarcat tool to extract them.
//
// This implements just the unzip logic from it to get around that.
// Obviously this only has impact on the plz repo itself, it's not a
// consideration in normal use.
package main

import (
	"fmt"
	"os"

	"github.com/thought-machine/please/tools/jarcat/unzip"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "x" {
		fmt.Fprintf(os.Stderr, "Usage: jarcat_unzip x <zipfile>\n")
		os.Exit(1)
	}
	if err := unzip.Extract(os.Args[2], ".", "", ""); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to extract: %s\n", err)
		os.Exit(1)
	}
}
