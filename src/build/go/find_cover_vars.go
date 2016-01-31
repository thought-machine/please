// Package gobuild contains utilities used by plz_go_test.
// It's split up mostly for ease of testing.
package gobuild

import (
	"bufio"
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	
	"github.com/op/go-logging"

	"buildgo"
	"output"
)

var log = logging.MustGetLogger("buildgo")

// FindCoverVars searches the given directory recursively to find all compiled packages in it.
// From these we extract any coverage variables that have been templated into them; unfortunately
// this isn't possible to examine dynamically using the reflect package.
func FindCoverVars(dir string, exclude []string) ([]string, err) {
	excludeMap := map[string]struct{}{}
	for _, e := range exclude {
		excludeMap[e] = struct{}{}
	}
	ret := []string{}
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if _, present := excludeMap[path]; present {
			return filepath.SkipDir
		} else if strings.HasSuffix(path, ".a") {
			vars, err := readPkgdef(path)
			if err != nil {
				return err
			}
			ret = append(ret, vars...)
		}
		return nil
	})
	return ret, err
}

// readPkgdef extracts the __.PKGDEF data from a Go object file.
// This is heavily based on go tool pack which does a similar thing.
func readPkgdef(file string) (vars []string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read from file, collecting header for __.PKGDEF.
	// The header is from the beginning of the file until a line
	// containing just "!". The first line must begin with "go object ".
	rbuf := bufio.NewReader(f)
	ret := []string{}
	for {
		line, err := rbuf.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if wbuf.Len() == 0 && !bytes.HasPrefix(line, []byte("go object ")) {
			// Slight alteration here; we cannot rely on all .a files being Go archives
			// (we might be linking against cgo libraries too). This is therefore nonfatal.
			log.Warning("%s isn't a Go object file, skipping", file)
			return []string{}, nil
		}
		if bytes.Equal(line, []byte("!\n")) {
			break
		}
		if index := bytes.Index(line, "var @\"\".GoCover"); index != -1 {
			line = line[8:]  // Strip the leading gunk
			ret = append(ret, string(line[:bytes.IndexRune(line, ' ')]))
		}
	}
	return ret, nil
}
