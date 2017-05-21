// Package tar implements a tarball writer for Please.
// This is not really dissimilar to the standard command-line tar utility,
// but we would like some of the GNU tar flags which we can't rely on for all
// platforms that we support, plus we'd like finer control over timestamps
// and directories.
package tar

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// mtime is the time we attach for the modification time of all files.
var mtime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// Write writes a tarball to output with all the files found in inputDir.
// If prefix is given the files are all placed into a single directory with that name.
// If compress is true the output will be gzip-compressed.
func Write(output, inputDir, prefix string, compress bool) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	if compress {
		w := gzip.NewWriter(f)
		defer w.Close()
		return write(w, inputDir, prefix)
	}
	return write(f, inputDir, prefix)
}

// write writes a tarball to the given writer with all the files found in inputDir.
// If prefix is given the files are all placed into a single directory with that name.
func write(w io.Writer, inputDir, prefix string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	return filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if info.IsDir() {
			return nil // ignore directories
		}
		hdr, err := tar.FileInfoHeader(info, "") // We don't write symlinks into plz-out/tmp, so the argument doesn't matter.
		if err != nil {
			return err
		}
		// Set name appropriately (recall that FileInfoHeader does not set the full path).
		if prefix != "" {
			hdr.Name = filepath.Join(prefix, hdr.Name)
		} else {
			hdr.Name = strings.TrimLeft(strings.TrimPrefix(path, inputDir), "/")
		}
		// Zero out all timestamps.
		hdr.ModTime = mtime
		hdr.AccessTime = mtime
		hdr.ChangeTime = mtime
		// Strip user/group ids.
		hdr.Uid = 0
		hdr.Gid = 0
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
