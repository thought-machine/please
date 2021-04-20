package install

import (
	"bufio"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/tools/please_go/install/exec"
	"github.com/thought-machine/please/tools/please_go/install/toolchain"
)

// PleaseGoInstall implements functionality similar to `go install` however it works with import configs to avoid a
// dependence on the GO_PATH, go.mod or other go build concepts.
type PleaseGoInstall struct {
	srcRoot      string
	moduleName   string
	importConfig string
	ldFlags      string
	outDir       string
	trimPath     string

	tc *toolchain.Toolchain

	compiledPackages map[string]bool
}

// New creates a new PleaseGoInstall
func New(srcRoot, moduleName, importConfig, ldFlags, goTool, ccTool, out, trimPath string) *PleaseGoInstall {
	return &PleaseGoInstall{
		srcRoot:      srcRoot,
		moduleName:   moduleName,
		importConfig: importConfig,
		ldFlags:      ldFlags,
		outDir:       out,
		trimPath:     trimPath,

		tc: &toolchain.Toolchain{
			CcTool: ccTool,
			GoTool: goTool,
			Exec:   &exec.OsExecutor{Stdout: os.Stdout, Stderr: os.Stderr},
		},
	}
}

// Install will compile the provided packages. Packages can be wildcards i.e. `foo/...` which compiles all packages
// under the directory tree of `{module}/foo`
func (install *PleaseGoInstall) Install(packages []string) error {
	if err := install.initBuildEnv(); err != nil {
		return err
	}
	if err := install.parseImportConfig(); err != nil {
		return err
	}

	for _, target := range packages {
		if !strings.HasPrefix(target, install.moduleName) {
			target = filepath.Join(install.moduleName, target)
		}
		if strings.HasSuffix(target, "/...") {
			importRoot := strings.TrimSuffix(target, "/...")
			err := install.compileAll(importRoot)
			if err != nil {
				return err
			}
		} else if err := install.compile([]string{}, target); err != nil {
			return fmt.Errorf("failed to compile %v: %w", target, err)
		}
	}
	return nil
}

// compileAll walks the provided directory looking for go packages to compile. Unlike compile(), this will skip any
// directories that contain no .go files for the current architecture.
func (install *PleaseGoInstall) compileAll(dir string) error {
	pkgRoot := install.pkgDir(dir)
	return filepath.Walk(pkgRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relativePackage := filepath.Dir(strings.TrimPrefix(path, pkgRoot))
			if err := install.compile([]string{}, filepath.Join(dir, relativePackage)); err != nil {
				switch err.(type) {
				case *build.NoGoError:
					// We might walk into a dir that has no .go files for the current arch. This shouldn't
					// be an error so we just eat this
					return nil
				default:
					return err
				}
			}
		} else if info.Name() == "testdata" {
			return filepath.SkipDir // Dirs named testdata are deemed not to contain buildable Go code.
		}
		return nil
	})
}

func (install *PleaseGoInstall) initBuildEnv() error {
	if err := install.tc.Exec.Exec("mkdir -p %s\n", filepath.Join(install.outDir, "bin")); err != nil {
		return err
	}
	return install.tc.Exec.Exec("touch %s", install.ldFlags)
}

// pkgDir returns the file path to the given target package
func (install *PleaseGoInstall) pkgDir(target string) string {
	p := strings.TrimPrefix(target, install.moduleName)
	return filepath.Join(install.srcRoot, p)
}

func (install *PleaseGoInstall) parseImportConfig() error {
	install.compiledPackages = map[string]bool{
		"unsafe": true, // Not sure how many other packages like this I need to handle
		"C":      true, // Pseudo-package for cgo symbols
	}

	if install.importConfig != "" {
		f, err := os.Open(install.importConfig)
		if err != nil {
			return fmt.Errorf("failed to open import config: %w", err)
		}
		defer f.Close()

		importCfg := bufio.NewScanner(f)
		for importCfg.Scan() {
			line := importCfg.Text()
			parts := strings.Split(strings.TrimPrefix(line, "packagefile "), "=")
			install.compiledPackages[parts[0]] = true
		}
	}
	return nil
}

func checkCycle(path []string, next string) ([]string, error) {
	for i, p := range path {
		if p == next {
			return nil, fmt.Errorf("package cycle detected: \n%s", strings.Join(append(path[i:], next), "\n ->"))
		}
	}

	return append(path, next), nil
}

func (install *PleaseGoInstall) compile(from []string, target string) error {
	if done := install.compiledPackages[target]; done {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Compiling package %s from %v\n", target, from)

	from, err := checkCycle(from, target)
	if err != nil {
		return err
	}

	pkgDir := install.pkgDir(target)
	// The package name can differ from the directory it lives in, in which case the parent directory is the one we want
	if _, err := os.Lstat(pkgDir); os.IsNotExist(err) {
		pkgDir = filepath.Dir(pkgDir)
	}

	// TODO(jpoole): is import vendor the correct thing to do here?
	pkg, err := build.ImportDir(pkgDir, build.ImportComment)
	if err != nil {
		return err
	}

	for _, i := range pkg.Imports {
		err := install.compile(from, i)
		if err != nil {
			if strings.Contains(err.Error(), "cannot find package") {
				// Go will fail to find this import and provide a much better message than we can
				continue
			}
			return err
		}
	}

	err = install.compilePackage(target, pkg)
	if err != nil {
		return err
	}
	install.compiledPackages[target] = true
	return nil
}

func (install *PleaseGoInstall) prepWorkdir(pkg *build.Package, workDir, out string) error {
	allSrcs := append(append(pkg.CFiles, pkg.GoFiles...), pkg.HFiles...)

	if err := install.tc.Exec.Exec("mkdir -p %s", workDir); err != nil {
		return err
	}
	if err := install.tc.Exec.Exec("mkdir -p %s", filepath.Dir(out)); err != nil {
		return err
	}
	return install.tc.Exec.Exec("ln %s %s", toolchain.FullPaths(allSrcs, pkg.Dir), workDir)
}

// outPath returns the path to the .a for a given package. Unlike go build, please_go install will always output to
// the same location regardless of if the package matches the package dir base e.g. example.com/foo will always produce
// example.com/foo/foo.a no matter what the package under there is named.
//
// We can get away with this because we don't compile tests so there must be exactly one package per directory.
func outPath(outDir, target string) string {
	dirName := filepath.Base(target)
	return filepath.Join(outDir, filepath.Dir(target), dirName, dirName+".a")
}

func (install *PleaseGoInstall) compilePackage(target string, pkg *build.Package) error {
	if len(pkg.GoFiles)+len(pkg.CgoFiles) == 0 {
		return nil
	}

	out := outPath(install.outDir, target)
	workDir := fmt.Sprintf("_build/%s", target)

	if err := install.prepWorkdir(pkg, workDir, out); err != nil {
		return fmt.Errorf("failed to prepare working directory for %s: %w", target, err)
	}

	goFiles := pkg.GoFiles

	var objFiles []string

	if len(pkg.CgoFiles) > 0 {
		cFiles := pkg.CFiles

		cgoGoFiles, cgoCFiles, err := install.tc.CGO(pkg.Dir, workDir, pkg.CgoFiles)
		if err != nil {
			return err
		}

		goFiles = append(goFiles, cgoGoFiles...)
		cFiles = append(cFiles, cgoCFiles...)

		cObjFiles, err := install.tc.CCompile(workDir, cFiles, pkg.CgoCFLAGS)
		if err != nil {
			return err
		}

		objFiles = append(objFiles, cObjFiles...)
	}

	if len(pkg.SFiles) > 0 {
		asmH, symabis, err := install.tc.Symabis(pkg.Dir, workDir, pkg.SFiles)
		if err != nil {
			return err
		}

		if err := install.tc.GoAsmCompile(workDir, install.importConfig, out, install.trimPath, goFiles, asmH, symabis); err != nil {
			return err
		}

		asmObjFiles, err := install.tc.Asm(pkg.Dir, workDir, install.trimPath, pkg.SFiles)
		if err != nil {
			return err
		}

		objFiles = append(objFiles, asmObjFiles...)
	} else {
		err := install.tc.GoCompile(workDir, install.importConfig, out, install.trimPath, goFiles)
		if err != nil {
			return err
		}
	}

	if len(objFiles) > 0 {
		err := install.tc.Pack(workDir, out, objFiles)
		if err != nil {
			return err
		}
	}

	if err := install.tc.Exec.Exec("echo \"packagefile %s=%s\" >> %s", target, out, install.importConfig); err != nil {
		return err
	}
	if len(pkg.CgoLDFLAGS) > 0 {
		if err := install.tc.Exec.Exec("echo -n \"%s\" >> %s", strings.Join(pkg.CgoLDFLAGS, " "), install.ldFlags); err != nil {
			return err
		}
	}

	if pkg.IsCommand() {
		filename := strings.TrimSuffix(filepath.Base(out), ".a")
		binName := filepath.Join(install.outDir, "bin", filename)

		return install.tc.Link(out, binName, install.importConfig, install.ldFlags)
	}
	return nil
}
