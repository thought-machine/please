// Package gotest contains utilities used by plz_go_test.
package gotest

import (
	"go/build"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("buildgo")

// A CoverVar is just a combination of package path and variable name
// for one of the templated-in coverage variables.
type CoverVar struct {
	Dir, ImportPath, ImportName, Var, File string
}

// FindCoverVars searches the given directory recursively to find all Go files with coverage variables.
func FindCoverVars(dir, importPath string, exclude, srcs []string) ([]CoverVar, error) {
	if dir == "" {
		return nil, nil
	}
	excludeMap := map[string]struct{}{}
	for _, e := range exclude {
		excludeMap[e] = struct{}{}
	}
	ret := []CoverVar{}

	err := fs.Walk(dir, func(name string, isDir bool) error {
		if _, present := excludeMap[name]; present {
			if isDir {
				return filepath.SkipDir
			}
		} else if strings.HasSuffix(name, ".a") && !strings.ContainsRune(path.Base(name), '#') {
			vars, err := findCoverVars(name, importPath, srcs)
			if err != nil {
				return err
			}
			for _, v := range vars {
				ret = append(ret, v)
			}
		}
		return nil
	})
	return ret, err
}

// findCoverVars scans a directory containing a .a file for any go files.
func findCoverVars(filepath, importPath string, srcs []string) ([]CoverVar, error) {
	dir, file := path.Split(filepath)
	dir = strings.TrimRight(dir, "/")
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	importPath = collapseFinalDir(strings.TrimSuffix(filepath, ".a"), importPath)
	ret := make([]CoverVar, 0, len(fi))
	for _, info := range fi {
		name := info.Name()
		if name != file && strings.HasSuffix(name, ".a") {
			log.Warning("multiple .a files in %s, can't determine coverage variables accurately", dir)
			return nil, nil
		} else if strings.HasSuffix(name, ".go") && !info.IsDir() && !contains(path.Join(dir, name), srcs) {
			if ok, err := build.Default.MatchFile(dir, name); ok && err == nil {
				// N.B. The scheme here must match what we do in go_rules.build_defs
				v := "GoCover_" + strings.Replace(name, ".", "_", -1)
				ret = append(ret, coverVar(dir, importPath, v))
			}
		}
	}
	return ret, nil
}

func contains(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

func coverVar(dir, importPath, v string) CoverVar {
	log.Info("Found cover variable: %s %s %s", dir, importPath, v)
	f := path.Join(dir, strings.TrimPrefix(v, "GoCover_"))
	if strings.HasSuffix(f, "_go") {
		f = f[:len(f)-3] + ".go"
	}
	return CoverVar{
		Dir:        dir,
		ImportPath: importPath,
		Var:        v,
		File:       f,
	}
}

// collapseFinalDir mimics what go does with import paths; if the final two components of
// the given path are the same (eg. "src/core/core") it collapses them into one ("src/core")
// Also if importPath is empty then it trims a leading src/
func collapseFinalDir(s, importPath string) string {
	if importPath == "" {
		s = strings.TrimPrefix(s, "src/")
	}
	if path.Base(path.Dir(s)) == path.Base(s) {
		return path.Dir(s)
	}
	return s
}
