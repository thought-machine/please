// Package buildgo contains utilities used by plz_go_test.
// It's split up mostly for ease of testing.
package buildgo

import (
	"bufio"
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("buildgo")

// A CoverVar is just a combination of package path and variable name
// for one of the templated-in coverage variables.
type CoverVar struct {
	Dir, ImportPath, ImportName, Package, Var, File string
}

// FindCoverVars searches the given directory recursively to find all compiled packages in it.
// From these we extract any coverage variables that have been templated into them; unfortunately
// this isn't possible to examine dynamically using the reflect package.
func FindCoverVars(dir string, exclude []string) ([]CoverVar, error) {
	if dir == "" {
		return nil, nil
	}
	excludeMap := map[string]struct{}{}
	for _, e := range exclude {
		excludeMap[e] = struct{}{}
	}
	ret := []CoverVar{}

	err := filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if _, present := excludeMap[name]; present {
			return filepath.SkipDir
		} else if strings.HasSuffix(name, ".a") {
			vars, err := readPkgdef(name)
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
func readPkgdef(file string) (vars []CoverVar, err error) {
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

	dir := path.Dir(file)
	importPath := collapseFinalDir(strings.TrimPrefix(strings.TrimSuffix(file, ".a"), "src/"))

	ret := []CoverVar{}
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
			pkg = string(line[8 : len(line)-1])
		}
		if index := bytes.Index(line, []byte("var @\"\".GoCover")); index != -1 {
			line = line[9:] // Strip the leading gunk
			v := string(line[:bytes.IndexByte(line, ' ')])
			f := path.Join(dir, strings.TrimPrefix(v, "GoCover_"))
			if strings.HasSuffix(f, "_go") {
				f = f[:len(f)-3] + ".go"
			}
			ret = append(ret, CoverVar{
				Dir:        dir,
				ImportPath: importPath,
				Package:    pkg,
				Var:        v,
				File:       f,
			})
		}
	}
	return ret, nil
}

// collapseFinalDir mimics what go does with import paths; if the final two components of
// the given path are the same (eg. "src/core/core") it collapses them into one ("src/core")
func collapseFinalDir(s string) string {
	if path.Base(path.Dir(s)) == path.Base(s) {
		return path.Dir(s)
	}
	return s
}
