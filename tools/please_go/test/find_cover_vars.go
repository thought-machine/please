// Package gotest contains utilities used by plz_go_test.
package test

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// A CoverVar is just a combination of package path and variable name
// for one of the templated-in coverage variables.
type CoverVar struct {
	Dir, ImportPath, ImportName, Var, File string
}

// replacer is used to replace characters in cover variables.
// The scheme here must match what we do in go_rules.build_defs
var replacer = strings.NewReplacer(
	".", "_",
	"-", "_",
)

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

	err := filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, present := excludeMap[name]; present {
			if info.IsDir() {
				return filepath.SkipDir
			}
		} else if strings.HasSuffix(name, ".a") && !strings.ContainsRune(path.Base(name), '#') {
			vars, err := findCoverVars(name, importPath, srcs)
			if err != nil {
				return err
			}
			ret = append(ret, vars...)
		}
		return nil
	})
	return ret, err
}

// findCoverVars scans a directory containing a .a file for any go files.
func findCoverVars(filepath, importPath string, srcs []string) ([]CoverVar, error) {
	dir, file := path.Split(filepath)
	dir = strings.TrimRight(dir, "/")
	if dir == "" {
		dir = "."
	}
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	importPath = collapseFinalDir(strings.TrimSuffix(filepath, ".a"), importPath)
	ret := make([]CoverVar, 0, len(fi))
	for _, info := range fi {
		name := info.Name()
		if name != file && strings.HasSuffix(name, ".a") {
			fmt.Fprintf(os.Stderr, "multiple .a files in %s, can't determine coverage variables accurately\n", dir)
			return nil, nil
		} else if strings.HasSuffix(name, ".go") && !info.IsDir() && !contains(path.Join(dir, name), srcs) {
			if ok, err := build.Default.MatchFile(dir, name); ok && err == nil {
				v := "GoCover_" + replacer.Replace(name)
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
	fmt.Fprintf(os.Stderr, "Found cover variable: %s %s %s\n", dir, importPath, v)
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
