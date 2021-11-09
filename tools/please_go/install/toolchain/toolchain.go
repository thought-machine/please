package toolchain

import (
	"fmt"
	"go/build"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/thought-machine/please/tools/please_go/install/exec"
)

var versionRegex = regexp.MustCompile("go version go1.([0-9]+).+")

type Toolchain struct {
	CcTool        string
	GoTool        string
	PkgConfigTool string

	Exec *exec.Executor
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
func (tc *Toolchain) CGO(sourceDir string, objectDir string, cFlags []string, cgoFiles []string) ([]string, []string, error) {
	goFiles := []string{"_cgo_gotypes.go"}
	cFiles := []string{"_cgo_export.c"}

	for _, cgoFile := range cgoFiles {
		goFiles = append(goFiles, strings.TrimSuffix(cgoFile, ".go")+".cgo1.go")
		cFiles = append(cFiles, strings.TrimSuffix(cgoFile, ".go")+".cgo2.c")
	}

	if err := tc.Exec.Run("(cd %s; %s tool cgo -objdir $OLDPWD/%s -- %s %s)", sourceDir, tc.GoTool, objectDir, strings.Join(cFlags, " "), paths(cgoFiles)); err != nil {
		return nil, nil, err
	}

	return goFiles, cFiles, nil
}

// GoCompile will compile the go sources and the generated .cgo1.go sources for the CGO files (if any)
func (tc *Toolchain) GoCompile(dir, importpath, importcfg, out, trimpath, embedCfg string, goFiles []string) error {
	if importpath != "" {
		importpath = fmt.Sprintf("-p %s", importpath)
	}
	if trimpath != "" {
		trimpath = fmt.Sprintf("-trimpath %s", trimpath)
	}
	if embedCfg != "" {
		embedCfg = fmt.Sprintf("-embedcfg %s", embedCfg)
	}
	return tc.Exec.Run("%s tool compile -pack %s %s %s -importcfg %s -o %s %s", tc.GoTool, importpath, trimpath, embedCfg, importcfg, out, FullPaths(goFiles, dir))
}

// GoAsmCompile will compile the go sources linking to the the abi symbols generated from symabis()
func (tc *Toolchain) GoAsmCompile(dir, importpath, importcfg, out, trimpath, embedCfg string, goFiles []string, asmH, symabys string) error {
	if importpath != "" {
		importpath = fmt.Sprintf("-p %s", importpath)
	}
	if trimpath != "" {
		trimpath = fmt.Sprintf("-trimpath %s", trimpath)
	}
	if embedCfg != "" {
		embedCfg = fmt.Sprintf("-embedcfg %s", embedCfg)
	}
	return tc.Exec.Run("%s tool compile -pack %s %s %s -importcfg %s -asmhdr %s -symabis %s -o %s %s", tc.GoTool, importpath, embedCfg, trimpath, importcfg, asmH, symabys, out, FullPaths(goFiles, dir))
}

// CCompile will compile c sources and return the object files that will be generated
func (tc *Toolchain) CCompile(dir string, cFiles, ccFiles, cFlags, ccFlags []string) ([]string, error) {
	objFiles := make([]string, 0, len(cFiles) + len(ccFiles))

	for _, cFile := range cFiles {
		objFiles = append(objFiles, strings.TrimSuffix(cFile, ".c") + ".o")
	}

	for _, ccFile := range ccFiles {
		objFiles = append(objFiles, strings.TrimSuffix(ccFile, ".cc") + ".o")
	}

	if len(cFiles) > 0 {
		err := tc.Exec.Run("(cd %s; %s -Wno-error -Wno-unused-parameter -c %s -I . _cgo_export.c %s)", dir, tc.CcTool, strings.Join(cFlags, " "), paths(cFiles))
		if err != nil {
			return nil, err
		}
	}
	if len(ccFiles) > 0 {
		err := tc.Exec.Run("(cd %s; %s -Wno-error -Wno-unused-parameter -c %s -I . %s)", dir, tc.CcTool, strings.Join(append(cFlags, ccFlags...), " "), paths(ccFiles))
		if err != nil {
			return nil, err
		}
	}
	return objFiles, nil
}

// Pack will add the object files in dir to the archive
func (tc *Toolchain) Pack(dir, archive string, objFiles []string) error {
	return tc.Exec.Run("%s tool pack r %s %s", tc.GoTool, archive, FullPaths(objFiles, dir))
}

// Link will link the archive into an executable
func (tc *Toolchain) Link(archive, out, importcfg string, ldFlags []string) error {
	return tc.Exec.Run("%s tool link -extld %s -extldflags \"%s\" -importcfg %s -o %s %s", tc.GoTool, tc.CcTool, strings.Join(ldFlags, " "), importcfg, out, archive)
}

// Symabis will generate the asm header as well as the abi symbol file for the provided asm files.
func (tc *Toolchain) Symabis(sourceDir, objectDir string, asmFiles []string) (string, string, error) {
	asmH := fmt.Sprintf("%s/go_asm.h", objectDir)
	symabis := fmt.Sprintf("%s/symabis", objectDir)

	// the gc Toolchain does this
	if err := tc.Exec.Run("touch %s", asmH); err != nil {
		return "", "", err
	}
	err := tc.Exec.Run("(cd %s; %s tool asm -I . -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -gensymabis -o $OLDPWD/%s %s)", sourceDir, tc.GoTool, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, symabis, paths(asmFiles))
	return asmH, symabis, err
}

// Asm will compile the asm files and return the objects that are generated
func (tc *Toolchain) Asm(sourceDir, objectDir, trimpath string, asmFiles []string) ([]string, error) {
	objFiles := make([]string, len(asmFiles))
	if trimpath != "" {
		trimpath = fmt.Sprintf("-trimpath %s", trimpath)
	}

	for i, asmFile := range asmFiles {
		objFile := strings.TrimSuffix(asmFile, ".s") + ".o"
		objFiles[i] = objFile

		err := tc.Exec.Run("(cd %s; %s tool asm %s -I . -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -o $OLDPWD/%s/%s %s)", sourceDir, tc.GoTool, trimpath, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, objectDir, objFile, asmFile)
		if err != nil {
			return nil, err
		}
	}

	return objFiles, nil
}

func (tc *Toolchain) GoMinorVersion() (int, error) {
	out, err := tc.Exec.CombinedOutput(tc.GoTool, "version")
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(versionRegex.FindSubmatch(out)[1]))
}

func (tc *Toolchain) PkgConfigCFlags(cfgs []string) ([]string, error) {
	return tc.pkgConfig("--cflags", cfgs)
}

func (tc *Toolchain) PkgConfigLDFlags(cfgs []string) ([]string, error) {
	return tc.pkgConfig("--libs", cfgs)
}

func (tc *Toolchain) pkgConfig(cmd string, cfgs []string) ([]string, error) {
	args := []string{cmd}
	out, err := tc.Exec.CombinedOutput(tc.PkgConfigTool, append(args, cfgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pkg configs %v: %w", cfgs, err)
	}
	return strings.Fields(string(out)), nil
}
