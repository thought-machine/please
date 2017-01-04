// Package buildgo contains utilities used by plz_go_test.
// It's split up mostly for ease of testing.
package buildgo

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("buildgo")

const arHeaderSize = 60

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
			for _, v := range vars {
				if strings.ContainsRune(v.ImportPath, '#') {
					log.Debug("Skipping cover variable with internal import path %s", v.ImportPath)
					continue
				}
				// Bit of a hack to avoid double-importing some cgo variables in rare cases.
				// TODO(pebers): This overgenerates a bit, find a better solution.
				dir, file := path.Split(v.ImportPath)
				cgoPath := path.Join(dir, "_"+file+"#c.a")
				if _, err := os.Stat(cgoPath); err == nil {
					log.Debug("Skipping cover variable which appears to have an associated cgo library: %s", cgoPath)
					continue
				}
				ret = append(ret, v)
			}
		}
		return nil
	})
	return ret, err
}

// readPkgdef extracts the __.PKGDEF data from a Go object file.
// This is heavily based on go tool pack which does a similar thing.
func readPkgdef(file string) ([]CoverVar, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rbuf := bufio.NewReader(f)
	// First lines contain some headers, make sure it's the right file then continue
	rbuf.ReadBytes('\n')
	line, err := rbuf.ReadBytes('\n')
	if !bytes.HasPrefix(line, []byte("__.PKGDEF")) {
		log.Warning("%s doesn't lead with a PKGDEF entry, skipping", file)
		return nil, nil
	}
	size, _ := strconv.ParseInt(string(bytes.TrimSpace(line[48:58])), 0, 64)

	// Read up to the line with the format marker
	for !bytes.HasPrefix(line, []byte("$$")) {
		line, err = rbuf.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
	}
	dir := path.Dir(file)
	importPath := collapseFinalDir(strings.TrimPrefix(strings.TrimSuffix(file, ".a"), "src/"))

	// 1.7+ have a binary object format instead
	if bytes.HasPrefix(line, []byte("$$B")) {
		contents := make([]byte, size-arHeaderSize)
		if _, err := io.ReadFull(rbuf, contents); err != nil {
			return nil, err
		}
		return readBinaryFormat(contents, dir, importPath)
	}
	return readTextFormat(rbuf, dir, importPath)
}

// readTextFormat reads variables from the older-style text format for .a files
func readTextFormat(rbuf *bufio.Reader, dir, importPath string) ([]CoverVar, error) {
	ret := []CoverVar{}
	pkg := ""
	var line []byte
	var err error
	for !bytes.HasPrefix(line, []byte("$$")) {
		line, err = rbuf.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if bytes.HasPrefix(line, []byte("package ")) {
			pkg = string(line[8 : len(line)-1])
		}
		if index := bytes.Index(line, []byte("var @\"\".GoCover")); index != -1 {
			line = line[9:] // Strip the leading gunk
			v := string(line[:bytes.IndexByte(line, ' ')])
			ret = append(ret, coverVar(dir, importPath, pkg, v))
		}
	}
	return ret, nil
}

// readBinaryFormat reads variables from the 1.7+ binary format for .a files.
// This is pretty fragile; there doesn't seem to be anything in the standard library to do
// this for us :(
func readBinaryFormat(contents []byte, dir, importPath string) ([]CoverVar, error) {
	contents = contents[7:] // Strip preamble
	u, i := binary.Varint(contents)
	i2 := i - int(u)
	pkg := string(contents[i:i2]) // string lengths are stored negative
	contents = contents[i2:]

	ret := []CoverVar{}
	prefix := []byte("GoCover_")
	for i := bytes.Index(contents, prefix); i != -1; i = bytes.Index(contents, prefix) {
		contents = contents[i:]
		end := bytes.IndexByte(contents, 0)
		v := bytes.TrimRight(bytes.TrimSuffix(contents[:end], []byte{15, 6}), "\x01\x02\x03\x04\x05\x06\x07\x08\x09")
		cv := coverVar(dir, importPath, pkg, string(v))
		if _, err := os.Stat(cv.File); os.IsNotExist(err) {
			log.Warning("Derived cover variable %s but file doesn't exist, skipping", cv.File)
		} else {
			ret = append(ret, cv)
		}
		contents = contents[end:]
	}
	return ret, nil
}

func coverVar(dir, importPath, pkg, v string) CoverVar {
	log.Info("Found cover variable: %s %s %s %s", dir, importPath, pkg, v)
	f := path.Join(dir, strings.TrimPrefix(v, "GoCover_"))
	if strings.HasSuffix(f, "_go") {
		f = f[:len(f)-3] + ".go"
	}
	return CoverVar{
		Dir:        dir,
		ImportPath: importPath,
		Package:    pkg,
		Var:        v,
		File:       f,
	}
}

// collapseFinalDir mimics what go does with import paths; if the final two components of
// the given path are the same (eg. "src/core/core") it collapses them into one ("src/core")
func collapseFinalDir(s string) string {
	if path.Base(path.Dir(s)) == path.Base(s) {
		return path.Dir(s)
	}
	return s
}
