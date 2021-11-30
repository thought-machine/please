package install

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thought-machine/please/tools/please_go/install/exec"
	"github.com/thought-machine/please/tools/please_go/install/toolchain"
	"github.com/thought-machine/please/tools/please_go_embed/embed"
)

const baseWorkDir = "_build"
const ldFlagsFile = "LD_FLAGS"

// PleaseGoInstall implements functionality similar to `go install` however it works with import configs to avoid a
// dependence on the GO_PATH, go.mod or other go build concepts.
type PleaseGoInstall struct {
	buildContext build.Context
	srcRoot      string
	moduleName   string
	importConfig string
	outDir       string
	trimPath     string

	additionalLDFlags string
	additionalCFlags  string

	tc *toolchain.Toolchain

	compiledPackages map[string]string

	// A set of flags we from pkg-config or #cgo comments
	collectedLdFlags []string
}

func (install *PleaseGoInstall) mustSetBuildContext(tags []string) {
	install.buildContext = build.Default
	install.buildContext.BuildTags = append(install.buildContext.BuildTags, tags...)

	version, err := install.tc.GoMinorVersion()
	if err != nil {
		log.Fatalf("failed to determine go version: %v", err)
	}

	install.buildContext.ReleaseTags = []string{}
	for i := 1; i <= version; i++ {
		install.buildContext.ReleaseTags = append(install.buildContext.ReleaseTags, "go1."+strconv.Itoa(i))
	}
}

// New creates a new PleaseGoInstall
func New(buildTags []string, srcRoot, moduleName, importConfig, ldFlags, cFlags, goTool, ccTool, pkgConfTool, out, trimPath string) *PleaseGoInstall {
	i := &PleaseGoInstall{
		srcRoot:      srcRoot,
		moduleName:   moduleName,
		importConfig: importConfig,
		outDir:       out,
		trimPath:     filepath.Join(trimPath, baseWorkDir),

		additionalLDFlags: ldFlags,
		additionalCFlags:  cFlags,

		tc: &toolchain.Toolchain{
			CcTool:        ccTool,
			GoTool:        goTool,
			PkgConfigTool: pkgConfTool,
			Exec:          &exec.Executor{Stdout: os.Stdout, Stderr: os.Stderr},
		},
	}
	i.mustSetBuildContext(buildTags)
	return i
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
		} else {
			if err := install.compile([]string{}, target); err != nil {
				return fmt.Errorf("failed to compile %v: %w", target, err)
			}

			pkg, err := install.importDir(target)
			if err != nil {
				panic(fmt.Sprintf("import dir failed after successful compilation: %v", err))
			}
			if pkg.IsCommand() {
				if err := install.linkPackage(target); err != nil {
					return fmt.Errorf("failed to link %v: %w", target, err)
				}
			}
		}
	}

	if err := install.writeLDFlags(); err != nil {
		return fmt.Errorf("failed to write ld flags: %w", err)
	}

	return nil
}

func (install *PleaseGoInstall) writeLDFlags() error {
	flagFile, err := os.Create(ldFlagsFile)
	if err != nil {
		return err
	}
	defer flagFile.Close()

	_, err = flagFile.WriteString(strings.Join(install.collectedLdFlags, " "))
	return err
}

func (install *PleaseGoInstall) linkPackage(target string) error {
	out := install.compiledPackages[target]
	filename := strings.TrimSuffix(filepath.Base(out), ".a")
	binName := filepath.Join(install.outDir, "bin", filename)

	return install.tc.Link(out, binName, install.importConfig, install.collectedLdFlags)
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
	if err := install.tc.Exec.Run("mkdir -p %s\n", filepath.Join(install.outDir, "bin")); err != nil {
		return err
	}
	return install.tc.Exec.Run("touch %s", ldFlagsFile)
}

// pkgDir returns the file path to the given target package
func (install *PleaseGoInstall) pkgDir(target string) string {
	p := strings.TrimPrefix(target, install.moduleName)
	return filepath.Join(install.srcRoot, p)
}

func (install *PleaseGoInstall) parseImportConfig() error {
	install.compiledPackages = map[string]string{
		"unsafe": "", // Not sure how many other packages like this I need to handle
		"C":      "", // Pseudo-package for cgo symbols
		"embed":  "", // Another psudo package
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
			install.compiledPackages[parts[0]] = parts[1]
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

func (install *PleaseGoInstall) importDir(target string) (*build.Package, error) {
	pkgDir := install.pkgDir(target)
	// TODO(jpoole): is this really the right thing to do? I think this is a please specific "bug"?
	// The package name can differ from the directory it lives in, in which case the parent directory is the one we want
	if _, err := os.Lstat(pkgDir); os.IsNotExist(err) {
		pkgDir = filepath.Dir(pkgDir)
	}
	return install.buildContext.ImportDir(filepath.Join(os.Getenv("TMP_DIR"), pkgDir), build.ImportComment)
}

func (install *PleaseGoInstall) compile(from []string, target string) error {
	if _, done := install.compiledPackages[target]; done {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Compiling package %s from %v\n", target, from)

	from, err := checkCycle(from, target)
	if err != nil {
		return err
	}

	pkg, err := install.importDir(target)
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
	return nil
}

func (install *PleaseGoInstall) prepWorkdir(pkg *build.Package, workDir, out string) error {
	allSrcs := append(append(append(pkg.CFiles, pkg.CXXFiles...), pkg.GoFiles...), pkg.HFiles...)

	if err := install.tc.Exec.Run("mkdir -p %s", workDir); err != nil {
		return err
	}
	if err := install.tc.Exec.Run("mkdir -p %s", filepath.Dir(out)); err != nil {
		return err
	}
	return install.tc.Exec.Run("ln %s %s", toolchain.FullPaths(allSrcs, pkg.Dir), workDir)
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

func writeEmbedConfig(pkg *build.Package, path string) error {
	cfg := &embed.Cfg{
		Patterns: map[string][]string{},
		Files:    map[string]string{},
	}

	if err := cfg.AddPackage(pkg); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0666)
}

func (install *PleaseGoInstall) compilePackage(target string, pkg *build.Package) error {
	if len(pkg.GoFiles)+len(pkg.CgoFiles) == 0 {
		return nil
	}

	out := outPath(install.outDir, target)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to find working directory: %s", err)
	}
	// We want `workDir` to maintain the same directory structure as `pkg.Dir`.
	// This will ensure that source file location point to the right place when debugging.
	relativePkgDir := strings.TrimPrefix(pkg.Dir, wd)
	workDir := filepath.Join(baseWorkDir, relativePkgDir)

	if err := install.prepWorkdir(pkg, workDir, out); err != nil {
		return fmt.Errorf("failed to prepare working directory for %s: %w", target, err)
	}

	goFiles := pkg.GoFiles

	var objFiles []string

	ldFlags := pkg.CgoLDFLAGS

	if len(pkg.CgoFiles) > 0 {
		cFlags := pkg.CgoCFLAGS

		if len(pkg.CgoPkgConfig) > 0 {
			pkgConfCFlags, err := install.tc.PkgConfigCFlags(pkg.CgoPkgConfig)
			if err != nil {
				return err
			}

			cFlags = append(cFlags, pkgConfCFlags...)

			pkgConfLDFlags, err := install.tc.PkgConfigLDFlags(pkg.CgoPkgConfig)
			if err != nil {
				return err
			}

			ldFlags = append(ldFlags, pkgConfLDFlags...)
			if len(pkgConfLDFlags) > 0 {
				fmt.Fprintf(os.Stderr, "------ ***** ------ ld flags for %s: %s\n", target, strings.Join(pkgConfLDFlags, " "))
			}
		}

		if f := install.additionalCFlags; f != "" {
			cFlags = append(cFlags, f)
		}

		cgoGoFiles, cgoCFiles, err := install.tc.CGO(pkg.Dir, workDir, cFlags, pkg.CgoFiles)
		if err != nil {
			return err
		}

		goFiles = append(goFiles, cgoGoFiles...)
		cFiles := append(pkg.CFiles, cgoCFiles...)

		cObjFiles, err := install.tc.CCompile(workDir, cFiles, pkg.CXXFiles, cFlags, pkg.CgoCXXFLAGS)
		if err != nil {
			return err
		}

		objFiles = append(objFiles, cObjFiles...)
	}

	embedConfig := ""
	if len(pkg.EmbedPatterns) > 0 {
		embedConfig = filepath.Join(workDir, "embed.cfg")
		if err := writeEmbedConfig(pkg, embedConfig); err != nil {
			return fmt.Errorf("failed to write embed config: %v", err)
		}
	}

	importPath := target
	if pkg.IsCommand() {
		importPath = "main"
	}

	if len(pkg.SFiles) > 0 {
		asmH, symabis, err := install.tc.Symabis(pkg.Dir, workDir, pkg.SFiles)
		if err != nil {
			return err
		}

		if err := install.tc.GoAsmCompile(workDir, importPath, install.importConfig, out, install.trimPath, embedConfig, goFiles, asmH, symabis); err != nil {
			return err
		}

		asmObjFiles, err := install.tc.Asm(pkg.Dir, workDir, install.trimPath, pkg.SFiles)
		if err != nil {
			return err
		}

		objFiles = append(objFiles, asmObjFiles...)
	} else {
		err := install.tc.GoCompile(workDir, importPath, install.importConfig, out, install.trimPath, embedConfig, goFiles)
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

	if err := install.tc.Exec.Run("echo \"packagefile %s=%s\" >> %s", target, out, install.importConfig); err != nil {
		return err
	}

	install.collectedLdFlags = append(install.collectedLdFlags, ldFlags...)

	install.compiledPackages[target] = out
	return nil
}
