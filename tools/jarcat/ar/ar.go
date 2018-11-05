// Package ar provides an ar file archiver.
package ar

import (
	"bufio"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/blakesmith/ar"

	"fs"
)

// mtime is the time we attach for the modification time of all files.
var mtime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// nobody is the usual uid / gid of the 'nobody' user.
const nobody = 65534

// Create creates a new ar archive from the given sources.
// If combine is true they are treated as existing ar files and combined.
// If rename is true the srcs are renamed as gcc would (i.e. the extension is replaced by .o).
func Create(srcs []string, out string, combine, rename bool) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	w := ar.NewWriter(bufio.NewWriter(f))
	if err := w.WriteGlobalHeader(); err != nil {
		return err
	}
	for _, src := range srcs {
		if rename {
			src = path.Base(src)
			if ext := path.Ext(src); ext != "" {
				src = src[:len(src)-len(ext)] + ".o"
			}
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
				}
				// Zero things out
				hdr.ModTime = mtime
				hdr.Uid = nobody
				hdr.Gid = nobody
				if err := w.WriteHeader(hdr); err != nil {
					return err
				} else if _, err := io.Copy(w, r); err != nil {
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
				Uid:     nobody,
				Gid:     nobody,
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
