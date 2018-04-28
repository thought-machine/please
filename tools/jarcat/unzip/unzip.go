// Package unzip implements unzipping for jarcat.
// We implement this to avoid needing a runtime dependency on unzip,
// which is not a profound package but not installed everywhere by default.
package unzip

import (
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"third_party/go/zip"
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
	r, err := zip.OpenReader(e.In)
	if err != nil {
		return err
	}
	defer r.Close()
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
