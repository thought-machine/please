// Package ar provides an ar file archiver.
package ar

import (
	"bufio"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/peterebden/ar"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("ar")

// mtime is the time we attach for the modification time of all files.
var mtime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// Create creates a new ar archive from the given sources.
// If combine is true they are treated as existing ar files and combined.
// If rename is true the srcs are renamed as gcc would (i.e. the extension is replaced by .o).
func Create(srcs []string, out string, combine, rename bool) error {
	// Rename the sources as gcc would.
	if rename {
		for i, src := range srcs {
			src = path.Base(src)
			if ext := path.Ext(src); ext != "" {
				src = src[:len(src)-len(ext)] + ".o"
			}
			srcs[i] = src
			log.Debug("renamed ar source to %s", src)
		}
	}

	log.Debug("Writing ar to %s", out)
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	defer bw.Flush()
	w := ar.NewWriter(bw)
	// Write BSD-style names on OSX, GNU-style ones on Linux
	if runtime.GOOS == "darwin" {
		if err := w.WriteGlobalHeader(); err != nil {
			return err
		}
	} else {
		if err := w.WriteGlobalHeaderForLongFiles(allSourceNames(srcs, combine)); err != nil {
			return err
		}
	}
	for _, src := range srcs {
		log.Debug("ar source file: %s", src)
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		if combine {
			// Read archive & write its contents in
			r := ar.NewReader(f)
			for {
				hdr, err := r.Next()
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				} else if hdr.Name == "/" || hdr.Name == "__.SYMDEF SORTED" || hdr.Name == "__.SYMDEF" {
					log.Debug("skipping symbol table")
					continue
				}
				// Zero things out
				hdr.ModTime = mtime
				hdr.Uid = 0
				hdr.Gid = 0
				log.Debug("copying '%s' in from %s", hdr.Name, src)
				if err := w.WriteHeader(hdr); err != nil {
					return err
				} else if _, err = io.Copy(w, r); err != nil {
					return err
				}
			}
		} else {
			// Write in individual file
			info, err := os.Lstat(src)
			if err != nil {
				return err
			}
			hdr := &ar.Header{
				Name:    src,
				ModTime: mtime,
				Mode:    int64(info.Mode()),
				Size:    info.Size(),
			}
			log.Debug("creating file %s", hdr.Name)
			if err := w.WriteHeader(hdr); err != nil {
				return err
			} else if _, err := io.Copy(w, f); err != nil {
				return err
			}
		}
		f.Close()
	}
	return nil
}

// Find finds all the .a files under the current directory and returns their names.
func Find() ([]string, error) {
	ret := []string{}
	return ret, fs.Walk(".", func(name string, isDir bool) error {
		if strings.HasSuffix(name, ".a") && !isDir {
			ret = append(ret, name)
		}
		return nil
	})
}

// allSourceNames returns the name of all source files that we will add to the archive.
func allSourceNames(srcs []string, combine bool) []string {
	if !combine {
		return srcs
	}
	ret := []string{}
	for _, src := range srcs {
		f, err := os.Open(src)
		if err == nil {
			r := ar.NewReader(f)
			for {
				hdr, err := r.Next()
				if err != nil {
					break
				}
				ret = append(ret, hdr.Name)
			}
		}
	}
	return ret
}
