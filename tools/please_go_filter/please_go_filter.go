// Package main implements please_go_filter, a tool for filtering go source files.
// This command parses source files and file names to determine whether they
// match the build constraints that you'd usually encounter with `go build`.
package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

var taglist = flag.String("tags", "", "Comma-separated list of extra build tags to apply as build constraints.")

func main() {
	flag.Parse()

	ctxt := build.Default
	ctxt.BuildTags = strings.Split(*taglist, ",")

	for _, f := range flag.Args() {
		dir, file := filepath.Split(f)

		// MatchFile skips _ prefixed files by default, assuming they're editor
		// temporary files - but we need to include cgo generated files.
		if strings.HasPrefix(file, "_cgo_") {
			fmt.Println(f)
			continue
		}

		ok, err := ctxt.MatchFile(dir, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking %v: %v\n", f, err)
			os.Exit(1)
		}

		if ok {
			fmt.Println(f)
		}
	}

	os.Exit(0)
}
