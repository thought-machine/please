package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type toolchain struct {

	ccTool string
	goTool string
}

func fullPaths(ps []string, dir string) string {
	fullPs := make([]string, len(ps))
	for i, p := range ps {
		fullPs[i] = filepath.Join(dir, p)
	}
	return paths(fullPs)
}

func paths(ps []string) string {
	return strings.Join(ps, " ") 
}

// cgo invokes go tool cgo to generate cgo sources and objects into the target directory
func (tc *toolchain) cgo(sourceDir string, objectDir string, cgoFiles []string) {
	fmt.Printf("(cd %s; %s tool cgo -objdir $OLDPWD/%s %s)\n", sourceDir, tc.goTool, objectDir, paths(cgoFiles))
}

// goCompile will compile the go sources and the generated .cgo2.go sources for the cgo files
func (tc *toolchain) goCompile(dir, importcfg, out string, goFiles, cgoFiles []string) {
	files := goFiles
	for _, cgo := range cgoFiles {
		files  = append(files, strings.TrimSuffix(cgo, ".go") + ".cgo1.go")
	}
	if len(cgoFiles) > 0 {
		files = append(files, "_cgo_gotypes.go")
	}
	fmt.Printf("%s tool compile -pack -importcfg %s -o %s %s\n", tc.goTool, importcfg, out, fullPaths(files, dir))
}

func (tc *toolchain) cCompile(dir string, cFiles, cgoFiles []string) {
	files := cFiles
	for _, cgo := range cgoFiles {
		files = append(files, strings.TrimSuffix(cgo, ".go") + ".cgo2.c")
	}
	fmt.Printf("(cd %s; %s -Wno-error -ldl -Wno-unused-parameter -c -I . _cgo_export.c %s)\n", dir, tc.ccTool, paths(files))
}

