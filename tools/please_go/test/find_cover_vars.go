// Package test contains utilities used by plz_go_test.
package test

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

var dontCollapseImportPaths = os.Getenv("FF_GO_DONT_COLLAPSE_IMPORT_PATHS")

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
func FindCoverVars(dir, importPath, testPackage string, external bool, exclude, srcs []string) ([]CoverVar, error) {
	if dir == "" {
		return nil, nil
	}
	excludeMap := map[string]struct{}{}
	for _, e := range exclude {
		excludeMap[e] = struct{}{}
	}
	var ret []CoverVar

	err := filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, present := excludeMap[name]; present {
			if info.IsDir() {
				return filepath.SkipDir
			}
		} else if strings.HasSuffix(name, ".a") && !strings.ContainsRune(filepath.Base(name), '#') {
			vars, err := findCoverVars(name, importPath, testPackage, external, srcs)
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
func findCoverVars(path, importPath, testPackage string, external bool, srcs []string) ([]CoverVar, error) {
	dir, file := filepath.Split(path)
	dir = strings.TrimRight(dir, "/")
	if dir == "" {
		dir = "."
	}

	packagePath := importPath
	if !external && dir == os.Getenv("PKG_DIR") {
		packagePath = testPackage
	} else if dir != "." {
		if dontCollapseImportPaths != "" {
			packagePath = toImportPath(dir, importPath)
		} else {
			packagePath = collapseFinalDir(strings.TrimSuffix(path, ".a"), importPath)
		}
	}

	fi, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	ret := make([]CoverVar, 0, len(fi))
	for _, info := range fi {
		name := info.Name()
		if name != file && strings.HasSuffix(name, ".a") {
			fmt.Fprintf(os.Stderr, "multiple .a files in %s, can't determine coverage variables accurately\n", dir)
			return nil, nil
		} else if strings.HasSuffix(name, ".go") && !info.IsDir() && !contains(filepath.Join(dir, name), srcs) {
			if ok, err := build.Default.MatchFile(dir, name); ok && err == nil {
				v := "GoCover_" + replacer.Replace(name)
				ret = append(ret, coverVar(dir, packagePath, v))
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
	f := filepath.Join(dir, strings.TrimPrefix(v, "GoCover_"))
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

// TODO(jpoole): delete this once v17 is released
// collapseFinalDir mimics what go does with import paths; if the final two components of
// the given path are the same (eg. "src/core/core") it collapses them into one ("src/core")
// Also if importPath is empty then it trims a leading src/
func collapseFinalDir(s, importPath string) string {
	if importPath == "" {
		s = strings.TrimPrefix(s, "src/")
	}
	if filepath.Base(filepath.Dir(s)) == filepath.Base(s) {
		s = filepath.Dir(s)
	}
	return filepath.Join(importPath, s)
}

// toImportPath converts a package directory path e.g. src/foo/bar to the import path for that package.
func toImportPath(s, modulePath string) string {
	if modulePath == "" {
		s = strings.TrimPrefix(s, "src/")
	}
	return filepath.Join(modulePath, s)
}
