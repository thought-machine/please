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

func mv(src, dest string) {
	if err := os.Rename(src, dest); err != nil {
		fmt.Fprintf(os.Stderr, "Error renaming source: %s", err)
		os.Exit(1)
	}
	fmt.Println(dest)
}

func main() {
	flag.Parse()

	ctxt := build.Default
	ctxt.BuildTags = strings.Split(*taglist, ",")

	pkg := os.Getenv("PKG")

	for _, f := range flag.Args() {
		dir, file := filepath.Split(f)
		dest := strings.TrimLeft(strings.TrimPrefix(f, pkg), "/")

		// MatchFile skips _ prefixed files by default, assuming they're editor
		// temporary files - but we need to include cgo generated files.
		if strings.HasPrefix(file, "_cgo_") {
			mv(f, dest)
			continue
		}

		ok, err := ctxt.MatchFile(dir, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking %v: %v\n", f, err)
			os.Exit(1)
		}

		if ok {
			mv(f, dest)
		}
	}
}
