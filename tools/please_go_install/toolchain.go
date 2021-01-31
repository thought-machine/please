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
func (tc *toolchain) cgo(sourceDir string, objectDir string, cgoFiles []string) ([]string, []string) {
	goFiles := []string{"_cgo_gotypes.go"}
	var cFiles []string

	for _, cgoFile := range cgoFiles {
		goFiles = append(goFiles, strings.TrimSuffix(cgoFile, ".go")+".cgo1.go")
		cFiles = append(cFiles, strings.TrimSuffix(cgoFile, ".go")+".cgo2.c")
	}

	fmt.Printf("(cd %s; %s tool cgo -objdir $OLDPWD/%s %s)\n", sourceDir, tc.goTool, objectDir, paths(cgoFiles))

	return goFiles, cFiles
}

// goCompile will compile the go sources and the generated .cgo1.go sources for the cgo files (if any)
func (tc *toolchain) goCompile(dir, importcfg, out string, goFiles []string) {
	fmt.Printf("%s tool compile -pack -importcfg %s -o %s %s\n", tc.goTool, importcfg, out, fullPaths(goFiles, dir))
}

// cCompile will compile c sources as well as the .cgo2.c files generated by go tool cgo (if any)
func (tc *toolchain) cCompile(dir string, cFiles []string) {
	fmt.Printf("(cd %s; %s -Wno-error -ldl -Wno-unused-parameter -c -I . _cgo_export.c %s)\n", dir, tc.ccTool, paths(cFiles))
}

func (tc *toolchain) pack(dir, out string, cFiles, cgoFiles []string) {
	objs := []string{"_cgo_export.o"}
	for _, o := range cFiles {
		objs = append(objs, strings.TrimSuffix(o, ".c")+".o")
	}

	for _, o := range cgoFiles {
		objs = append(objs, strings.TrimSuffix(o, ".go")+".cgo2.o")
	}

	if len(objs) == 1 {
		return
	}

	fmt.Printf("%s tool pack r %s %s\n", tc.goTool, out, fullPaths(objs, dir))
}

func (tc *toolchain) link(archive, out string) {
	fmt.Printf("%s tool link -importcfg %s -o %s %s", opts.GoTool, opts.ImportConfig, out, archive)
}
