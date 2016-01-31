// Package buildgo contains utilities used by plz_go_test.
// It's split up mostly for ease of testing.
package buildgo

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("buildgo")

// FindCoverVars searches the given directory recursively to find all compiled packages in it.
// From these we extract any coverage variables that have been templated into them; unfortunately
// this isn't possible to examine dynamically using the reflect package.
func FindCoverVars(dir string, exclude []string) ([]string, error) {
	if dir == "" {
		return nil, nil
	}
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

	rbuf := bufio.NewReader(f)
	// First lines contain some headers, make sure it's the right file then continue
	rbuf.ReadBytes('\n')
	line, _ := rbuf.ReadBytes('\n')
	if !bytes.HasPrefix(line, []byte("__.PKGDEF")) {
		log.Warning("%s doesn't lead with a PKGDEF entry, skipping", file)
		return nil, nil
	}
	
	ret := []string{}
	pkg := ""
	for {
		line, err := rbuf.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if bytes.Equal(line, []byte("!\n")) {
			break
		}
		if bytes.HasPrefix(line, []byte("package ")) {
			pkg = string(line[8:len(line)-1])
		}
		if index := bytes.Index(line, []byte("var @\"\".GoCover")); index != -1 {
			line = line[8:]  // Strip the leading gunk
			ret = append(ret, pkg + string(line[:bytes.IndexByte(line, ' ')]))
		}
	}
	return ret, nil
}
