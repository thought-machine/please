package main

import (
	"fmt"
	"go/build"
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
	cFiles := []string{"_cgo_export.c"}

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

// goAsmCompile will compile the go sources linking to the the abi symbols generated from symabis()
func (tc *toolchain) goAsmCompile(dir, importcfg, out string, goFiles []string, asmH, symabys string) {
	fmt.Printf("%s tool compile -pack -importcfg %s -asmhdr %s -symabis %s -o %s %s\n", tc.goTool, importcfg, asmH, symabys, out, fullPaths(goFiles, dir))
}

// cCompile will compile c sources and return the object files that will be generated
func (tc *toolchain) cCompile(dir string, cFiles []string, cFlags []string) []string {
	objFiles := make([]string, len(cFiles))

	for i, cFile := range cFiles {
		objFiles[i] = strings.TrimSuffix(cFile, ".c") + ".o"
	}

	fmt.Printf("(cd %s; %s -Wno-error -Wno-unused-parameter -c %s -I . _cgo_export.c %s)\n", dir, tc.ccTool, strings.Join(cFlags, " "), paths(cFiles))

	return objFiles
}

// pack will add the object files in dir to the archive
func (tc *toolchain) pack(dir, archive string, objFiles []string) {
	fmt.Printf("%s tool pack r %s %s\n", tc.goTool, archive, fullPaths(objFiles, dir))
}

// link will link the archive into an executable
func (tc *toolchain) link(archive, out string) {
	fmt.Printf("%s tool link -importcfg %s -o %s %s", opts.GoTool, opts.ImportConfig, out, archive)
}

// symabis will generate the asm header as well as the abi symbol file for the provided asm files.
func (tc *toolchain) symabis(sourceDir, objectDir string, asmFiles []string) (string, string) {
	asmH := fmt.Sprintf("%s/go_asm.h", objectDir)
	symabis := fmt.Sprintf("%s/symabis", objectDir)

	// the gc toolchain does this
	fmt.Printf("touch %s\n", asmH)

	fmt.Printf("(cd %s; %s tool asm -I . -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -gensymabis -o $OLDPWD/%s %s)\n", sourceDir, opts.GoTool, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, symabis, paths(asmFiles))

	return asmH, symabis
}

// asm will compile the asm files and return the objects that are generated
func (tc *toolchain) asm(sourceDir, objectDir string, asmFiles []string) []string {
	objFiles := make([]string, len(asmFiles))

	for i, asmFile := range asmFiles {
		objFile := strings.TrimSuffix(asmFile, ".s") + ".o"
		objFiles[i] = objFile

		fmt.Printf("(cd %s; %s tool asm -I . -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -o $OLDPWD/%s/%s %s)\n", sourceDir, opts.GoTool, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, objectDir, objFile, asmFile)
	}

	return objFiles
}
