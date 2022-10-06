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

func paths(ps []string) string {
	return strings.Join(ps, " ")
}

// CGO invokes go tool cgo to generate cgo sources in the target's object directory
func (tc *Toolchain) CGO(sourceDir string, objectDir string, cFlags []string, cgoFiles []string) ([]string, []string, error) {
	// Looking at `go build -work -n -a`, there's also `_cgo_main.c` that gets taken into account,
	// which results in a couple more commands being run.
	// Although we seem to ignore this file, it doesn't seem to cause things to break so far, but
	// leaving this note here for future reference.
	goFiles := []string{filepath.Join(objectDir, "_cgo_gotypes.go")}
	cFiles := []string{filepath.Join(objectDir, "_cgo_export.c")}

	for _, cgoFile := range cgoFiles {
		baseGoFile := strings.TrimSuffix(filepath.Base(cgoFile), ".go") + ".cgo1.go"
		baseCFile := strings.TrimSuffix(filepath.Base(cgoFile), ".go") + ".cgo2.c"

		goFiles = append(goFiles, filepath.Join(objectDir, baseGoFile))
		cFiles = append(cFiles, filepath.Join(objectDir, baseCFile))
	}

	// Although we don't set the `-importpath` flag here, it shows up in `go build -work -n -a`.
	// It doesn't seem to cause things to break without it so far, but leaving this note here for future reference.
	if err := tc.Exec.Run("(cd %s; %s tool cgo -objdir %s -- -I %s %s %s)", sourceDir, tc.GoTool, objectDir, objectDir, strings.Join(cFlags, " "), paths(cgoFiles)); err != nil {
		return nil, nil, err
	}

	return goFiles, cFiles, nil
}

// GoCompile will compile the go sources and the generated .cgo1.go sources for the CGO files (if any)
func (tc *Toolchain) GoCompile(sourceDir, importpath, importcfg, out, trimpath, embedCfg string, goFiles []string) error {
	if importpath != "" {
		importpath = fmt.Sprintf("-p %s", importpath)
	}
	if trimpath != "" {
		trimpath = fmt.Sprintf("-trimpath %s", trimpath)
	}
	if embedCfg != "" {
		embedCfg = fmt.Sprintf("-embedcfg %s", embedCfg)
	}
	return tc.Exec.Run("%s tool compile -pack %s %s %s -importcfg %s -o %s %s", tc.GoTool, importpath, trimpath, embedCfg, importcfg, out, paths(goFiles))
}

// GoAsmCompile will compile the go sources linking to the the abi symbols generated from symabis()
func (tc *Toolchain) GoAsmCompile(importpath, importcfg, out, trimpath, embedCfg string, goFiles []string, asmH, symabys string) error {
	if importpath != "" {
		importpath = fmt.Sprintf("-p %s", importpath)
	}
	if trimpath != "" {
		trimpath = fmt.Sprintf("-trimpath %s", trimpath)
	}
	if embedCfg != "" {
		embedCfg = fmt.Sprintf("-embedcfg %s", embedCfg)
	}
	return tc.Exec.Run("%s tool compile -pack %s %s %s -importcfg %s -asmhdr %s -symabis %s -o %s %s", tc.GoTool, importpath, embedCfg, trimpath, importcfg, asmH, symabys, out, paths(goFiles))
}

// CCompile will compile C/CXX sources and return the object files that will be generated
func (tc *Toolchain) CCompile(sourceDir, objectDir string, ccFiles, ccFlags []string) ([]string, error) {
	objFiles := make([]string, len(ccFiles))

	for i, ccFile := range ccFiles {
		baseObjFile := strings.TrimSuffix(filepath.Base(ccFile), filepath.Ext(ccFile)) + ".o"
		objFiles[i] = filepath.Join(objectDir, baseObjFile)

		if err := tc.Exec.Run("(cd %s; %s -Wno-error -Wno-unused-parameter -c %s -I . -o %s %s)", sourceDir, tc.CcTool, strings.Join(ccFlags, " "), objFiles[i], ccFile); err != nil {
			return nil, err
		}
	}

	return objFiles, nil
}

// Pack will add the object files in dir to the archive
func (tc *Toolchain) Pack(dir, archive string, objFiles []string) error {
	return tc.Exec.Run("%s tool pack r %s %s", tc.GoTool, archive, paths(objFiles))
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

	// Although we don't set both `-p` and `-trimpath` flag here, they show up in `go build -work -n -a`.
	// It doesn't seem to cause things to break without them so far, but leaving this note here for future reference.
	err := tc.Exec.Run("(cd %s; %s tool asm -I %s -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -gensymabis -o %s %s)", sourceDir, tc.GoTool, objectDir, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, symabis, paths(asmFiles))

	return asmH, symabis, err
}

// Asm will compile the asm files and return the objects that are generated
func (tc *Toolchain) Asm(importpath, sourceDir, objectDir, trimpath string, asmFiles []string) ([]string, error) {
	if importpath != "" {
		importpath = fmt.Sprintf("-p %s", importpath)
	}
	if trimpath != "" {
		trimpath = fmt.Sprintf("-trimpath %s", trimpath)
	}

	objFiles := make([]string, len(asmFiles))

	for i, asmFile := range asmFiles {
		baseObjFile := strings.TrimSuffix(filepath.Base(asmFile), ".s") + ".o"
		objFiles[i] = filepath.Join(objectDir, baseObjFile)

		err := tc.Exec.Run("(cd %s; %s tool asm %s %s -I %s -I %s/pkg/include -D GOOS_%s -D GOARCH_%s -o %s %s)", sourceDir, tc.GoTool, importpath, trimpath, objectDir, build.Default.GOROOT, build.Default.GOOS, build.Default.GOARCH, objFiles[i], asmFile)
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
