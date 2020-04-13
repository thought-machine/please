// Package main implements an extremely cut-down version of jarcat
// which is used to break a circular dependency; jarcat depends on
// several third-party libraries, and we use the jarcat tool to extract them.
//
// This implements minimal unzip logic that is command-line compatible with that
// subcommand.
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "x" {
		fmt.Fprintf(os.Stderr, "Usage: jarcat_unzip x <zipfile>\n")
		os.Exit(1)
	}
	if err := extract(os.Args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to extract: %s\n", err)
		os.Exit(1)
	}
}

func extract(zipfile string) error {
	zf, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}
	defer zf.Close()
	for _, f := range zf.File {
		if f.Mode()&os.ModeDir == 0 {
			if err := extractFile(f); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractFile(f *zip.File) error {
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	if err := os.MkdirAll(path.Dir(f.Name), os.ModeDir|0755); err != nil {
		return err
	}
	o, err := os.OpenFile(f.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer o.Close()
	_, err = io.Copy(o, r)
	return err
}
