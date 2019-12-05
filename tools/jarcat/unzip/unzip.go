// Package unzip implements unzipping for jarcat.
// We implement this to avoid needing a runtime dependency on unzip,
// which is not a profound package but not installed everywhere by default.
package unzip

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/ulikunitz/xz"

	"github.com/thought-machine/please/third_party/go/zip"
)

// concurrency controls the maximum level of concurrency we'll allow.
const concurrency = 4

// Extract extracts the contents of the given zipfile.
func Extract(in, out, file, prefix string) error {
	e := extractor{
		In:     in,
		Out:    out,
		File:   file,
		Prefix: prefix,
		dirs:   map[string]struct{}{},
	}
	return e.Extract()
}

// An extractor extracts a single zipfile.
type extractor struct {
	In     string
	Out    string
	File   string
	Prefix string
	dirs   map[string]struct{}
	mutex  sync.Mutex
	wg     sync.WaitGroup
	err    error
}

func (e *extractor) Extract() error {
	if r, err := zip.OpenReader(e.In); err == nil {
		defer r.Close()
		return e.extractZip(r)
	}
	f, err := os.Open(e.In)
	if err != nil {
		return err
	}
	defer f.Close()
	if r, err := gzip.NewReader(f); err == nil {
		if err := e.extractTar(r); err != nil {
			return err
		}
		return r.Close()
	}
	// Reset back to the start of the file and try xz
	f.Seek(0, os.SEEK_SET)
	if r, err := xz.NewReader(f); err == nil {
		return e.extractTar(r)
	}
	// Reset again and try bzip2
	f.Seek(0, os.SEEK_SET)
	if err := e.extractTar(bzip2.NewReader(f)); err == nil || !isStructuralError(err) {
		return err
	}
	// Assume uncompressed.
	f.Seek(0, os.SEEK_SET)
	return e.extractTar(f)
}

func isStructuralError(err error) bool {
	_, ok := err.(bzip2.StructuralError)
	return ok
}

func (e *extractor) extractTar(f io.Reader) error {
	r := tar.NewReader(f)
	for {
		hdr, err := r.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		out := path.Join(e.Out, strings.TrimLeft(strings.TrimPrefix(hdr.Name, e.Prefix), "/"))
		if err := e.makeDir(out); err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(out, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if f, err := os.Create(out); err != nil {
				return err
			} else if _, err := io.Copy(f, r); err != nil {
				return err
			} else if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(path.Join(e.Out, hdr.Linkname), out); err != nil {
				return err
			}
		default:
			fmt.Fprintf(os.Stderr, "Unhandled file type %d for %s", hdr.Typeflag, hdr.Name)
		}
	}
}

func (e *extractor) extractZip(r *zip.ReadCloser) error {
	ch := make(chan *zip.File, 100)
	for i := 0; i < concurrency; i++ {
		go e.consume(ch)
	}
	for _, f := range r.File {
		if e.File != "" && f.Name != e.File {
			continue
		}
		// This will mean that empty directories aren't created. We might need to fix that at some point.
		if f.Mode()&os.ModeDir == 0 {
			e.wg.Add(1)
			ch <- f
		}
	}
	e.wg.Wait()
	close(ch)
	return e.err
}

func (e *extractor) consume(ch <-chan *zip.File) {
	for f := range ch {
		if err := e.extractFile(f); err != nil {
			e.err = err
		}
		e.wg.Done()
	}
}

func (e *extractor) extractFile(f *zip.File) error {
	if e.Prefix != "" {
		if !strings.HasPrefix(f.Name, e.Prefix) {
			return nil
		}
		f.Name = strings.TrimLeft(strings.TrimPrefix(f.Name, e.Prefix), "/")
	}
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	out := path.Join(e.Out, f.Name)
	if e.File != "" {
		out = e.Out
	}
	if err := e.makeDir(out); err != nil {
		return err
	}
	o, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, f.Mode())
	if err != nil {
		return err
	}
	defer o.Close()
	_, err = io.Copy(o, r)
	return err
}

func (e *extractor) makeDir(filename string) error {
	dir := path.Dir(filename)
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if _, present := e.dirs[dir]; !present {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		e.dirs[dir] = struct{}{}
	}
	return nil
}
