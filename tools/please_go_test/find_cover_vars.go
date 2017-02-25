// Package buildgo contains utilities used by plz_go_test.
// It's split up mostly for ease of testing.
package buildgo

import (
	"io/ioutil"
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
	Dir, ImportPath, ImportName, Var, File string
}

// FindCoverVars searches the given directory recursively to find all Go files with coverage variables.
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
			vars, err := findCoverVars(name)
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
func findCoverVars(filepath string) ([]CoverVar, error) {
	dir := path.Dir(filepath)
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	importPath := collapseFinalDir(strings.TrimPrefix(strings.TrimSuffix(filepath, ".a"), "src/"))
	ret := make([]CoverVar, 0, len(fi))
	for _, info := range fi {
		if strings.HasSuffix(info.Name(), ".go") && !info.IsDir() {
			// N.B. The scheme here must match what we do in go_rules.build_defs
			v := "GoCover_" + strings.Replace(info.Name(), ".", "_", -1)
			ret = append(ret, coverVar(dir, importPath, v))
		}
	}
	return ret, nil
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
func collapseFinalDir(s string) string {
	if path.Base(path.Dir(s)) == path.Base(s) {
		return path.Dir(s)
	}
	return s
}
