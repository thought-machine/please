package toolchain

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/tools/please_go_install/exec"
)

type Toolchain struct {
	CcTool string
	GoTool string

	Exec exec.Executor
}

func FullPaths(ps []string, dir string) string {
	fullPs := make([]string, len(ps))
	for i, p := range ps {
		fullPs[i] = filepath.Join(dir, p)
	}
	return paths(fullPs)
}

func paths(ps []string) string {
	return strings.Join(ps, " ")
}

// CGO invokes go tool cgo to generate cgo sources in the target directory
func (tc *Toolchain) CGO(sourceDir string, objectDir string, cgoFiles []string) ([]string, []string) {
	goFiles := []string{"_cgo_gotypes.go"}
	cFiles := []string{"_cgo_export.c"}

	for _, cgoFile := range cgoFiles {
		goFiles = append(goFiles, strings.TrimSuffix(cgoFile, ".go")+".cgo1.go")
		cFiles = append(cFiles, strings.TrimSuffix(cgoFile, ".go")+".cgo2.c")
	}

	tc.Exec.Exec("(cd %s; %s tool cgo -objdir $OLDPWD/%s %s)", sourceDir, tc.GoTool, objectDir, paths(cgoFiles))

	return goFiles, cFiles
}

// GoCompile will compile the go sources and the generated .cgo1.go sources for the CGO files (if any)
func (tc *Toolchain) GoCompile(dir, importcfg, out string, goFiles []string) {
	tc.Exec.Exec("%s tool compile -pack -importcfg %s -o %s %s", tc.GoTool, importcfg, out, FullPaths(goFiles, dir))
}

// GoAsmCompile will compile the go sources linking to the the abi symbols generated from symabis()
func (tc *Toolchain) GoAsmCompile(dir, importcfg, out string, goFiles []string, asmH, symabys string) {
	tc.Exec.Exec("%s tool compile -pack -importcfg %s -asmhdr %s -symabis %s -o %s %s", tc.GoTool, importcfg, asmH, symabys, out, FullPaths(goFiles, dir))
}

// CCompile will compile c sources and return the object files that will be generated
func (tc *Toolchain) CCompile(dir string, cFiles []string, cFlags []string) []string {
	objFiles := make([]string, len(cFiles))

	for i, cFile := range cFiles {
		objFiles[i] = strings.TrimSuffix(cFile, ".c") + ".o"
	}

	tc.Exec.Exec("(cd %s; %s -Wno-error -Wno-unused-parameter -c %s -I . _cgo_export.c %s)", dir, tc.CcTool, strings.Join(cFlags, " "), paths(cFiles))
	return objFiles
}

// Pack will add the object files in dir to the archive
func (tc *Toolchain) Pack(dir, archive string, objFiles []string) {
	tc.Exec.Exec("%s tool pack r %s %s", tc.GoTool, archive, FullPaths(objFiles, dir))
}

// Link will link the archive into an executable
func (tc *Toolchain) Link(archive, out, importcfg, flags string) {
	tc.Exec.Exec("%s tool link -extld %s -extldflags \"$(cat %s)\" -importcfg %s -o %s %s", tc.GoTool, tc.CcTool, flags, importcfg, out, archive)
}

// Symabis will generate the asm header as well as the abi symbol file for the provided asm files.
func (tc *Toolchain) Symabis(sourceDir, objectDir string, asmFiles []string) (string, string) {
	asmH := fmt.Sprintf("%s/go_asm.h", objectDir)
	symabis := fmt.Sprintf("%s/symabis", objectDir)

	// the gc Toolchain does this
	tc.Exec.Exec("touch %s", asmH)
	tc.Exec.Exec("(cd %s; %s tool asm -I . -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -gensymabis -o $OLDPWD/%s %s)", sourceDir, tc.GoTool, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, symabis, paths(asmFiles))

	return asmH, symabis
}

// Asm will compile the asm files and return the objects that are generated
func (tc *Toolchain) Asm(sourceDir, objectDir string, asmFiles []string) []string {
	objFiles := make([]string, len(asmFiles))

	for i, asmFile := range asmFiles {
		objFile := strings.TrimSuffix(asmFile, ".s") + ".o"
		objFiles[i] = objFile

		tc.Exec.Exec("(cd %s; %s tool asm -I . -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -o $OLDPWD/%s/%s %s)", sourceDir, tc.GoTool, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, objectDir, objFile, asmFile)
	}

	return objFiles
}
