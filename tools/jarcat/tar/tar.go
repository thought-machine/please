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

	"github.com/ulikunitz/xz"
)

// mtime is the time we attach for the modification time of all files.
var mtime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// nobody is the usual uid / gid of the 'nobody' user.
const nobody = 65534

// Write writes a tarball to output with all the files found in inputDir.
// If prefix is given the files are all placed into a single directory with that name.
// If compress is true the output will be gzip-compressed.
func Write(output string, srcs []string, prefix string, gzcompress, xzcompress, flatten bool, stripPrefix string) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	if xzcompress {
		w, err := xz.NewWriter(f)
		if err != nil {
			return err
		}
		defer w.Close()
		return write(w, output, srcs, prefix, flatten, stripPrefix)
	} else if gzcompress {
		w := gzip.NewWriter(f)
		defer w.Close()
		return write(w, output, srcs, prefix, flatten, stripPrefix)
	}
	return write(f, output, srcs, prefix, flatten, stripPrefix)
}

// write writes a tarball to the given writer with all the files found in inputDir.
// If prefix is given the files are all placed into a single directory with that name.
func write(w io.Writer, output string, srcs []string, prefix string, flatten bool, stripPrefix string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, src := range srcs {
		strip := stripPrefix
		if flatten {
			strip = filepath.Dir(src)
		}
		if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			} else if info.IsDir() {
				return nil // ignore directories
			} else if abs, _ := filepath.Abs(path); abs == output {
				return nil // don't write the output tarball into itself :)
			}
			hdr, err := tar.FileInfoHeader(info, "") // We don't write symlinks into plz-out/tmp, so the argument doesn't matter.
			if err != nil {
				return err
			}
			// Set name appropriately (recall that FileInfoHeader does not set the full path).
			hdr.Name = strings.TrimLeft(strings.TrimPrefix(path, strip), "/")
			if prefix != "" {
				hdr.Name = filepath.Join(prefix, hdr.Name)
			}
			// Zero out all timestamps.
			hdr.ModTime = mtime
			hdr.AccessTime = mtime
			hdr.ChangeTime = mtime
			// Strip user/group ids.
			hdr.Uid = nobody
			hdr.Gid = nobody
			hdr.Uname = "nobody"
			hdr.Gname = "nobody"
			// Setting the user/group write bits helps consistency of output.
			hdr.Mode |= 0220
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
		}); err != nil {
			return err
		}
	}
	return nil
}
