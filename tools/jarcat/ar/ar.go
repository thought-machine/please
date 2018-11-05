// Package ar provides an ar file archiver.
package ar

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/blakesmith/ar"
	"gopkg.in/op/go-logging.v1"

	"fs"
)

var log = logging.MustGetLogger("ar")

// mtime is the time we attach for the modification time of all files.
var mtime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// Create creates a new ar archive from the given sources.
// If combine is true they are treated as existing ar files and combined.
// If rename is true the srcs are renamed as gcc would (i.e. the extension is replaced by .o).
func Create(srcs []string, out string, combine, rename bool) error {
	log.Debug("Writing ar to %s", out)
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	defer bw.Flush()
	w := ar.NewWriter(bw)
	if err := w.WriteGlobalHeader(); err != nil {
		return err
	}
	for _, src := range srcs {
		log.Debug("ar source file: %s", src)
		if rename {
			src = path.Base(src)
			if ext := path.Ext(src); ext != "" {
				src = src[:len(src)-len(ext)] + ".o"
			}
			log.Debug("renamed ar source to %s", src)
		}
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
				} else if hdr.Name == "/" {
					log.Debug("skipping symbol table")
					continue
				}
				// Zero things out
				hdr.ModTime = mtime
				hdr.Uid = 0
				hdr.Gid = 0

				copyFile := func(r io.Reader, name string) error {
					log.Debug("copying '%s' in from %s", name, src)
					if err := w.WriteHeader(hdr); err != nil {
						return err
					}
					_, err := io.Copy(w, r)
					return err
				}

				if !strings.HasPrefix(hdr.Name, "#1/") {
					// Normal copy
					if err := copyFile(r, hdr.Name); err != nil {
						return err
					}
					continue
				}

				// BSD ar stores the name as a prefix to the data section.
				b, err := ioutil.ReadAll(r)
				if err != nil {
					return err
				}
				l, err := strconv.Atoi(hdr.Name[3:])
				if err != nil {
					return err
				}
				name := string(bytes.TrimRightFunc(b[:l], func(r rune) bool { return r == 0 }))
				if name == "__.SYMDEF SORTED" || name == "__.SYMDEF" {
					log.Debug("skipping BSD symbol table")
					continue
				}
				if err := copyFile(bytes.NewReader(b), name); err != nil {
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
