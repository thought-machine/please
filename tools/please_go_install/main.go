package main

import (
	"bufio"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/go-flags"

	"github.com/thought-machine/please/tools/please_go_install/exec"
	"github.com/thought-machine/please/tools/please_go_install/toolchain"
)

var opts = struct {
	Usage string

	SrcRoot      string `short:"r" long:"src_root" description:"The src root of the module to inspect" default:"."`
	ModuleName   string `short:"n" long:"module_name" description:"The name of the module"`
	ImportConfig string `short:"i" long:"importcfg" description:"The import config for the modules dependencies"`
	LDFlags      string `short:"l" long:"ld_flags" description:"The file to write linker flags to" default:"LD_FLAGS"`
	GoTool       string `short:"g" long:"go_tool" description:"The location of the go binary"`
	CCTool       string `short:"c" long:"cc_tool" description:"The c compiler to use"`
	Out          string `short:"o" long:"out" description:"The output directory to put compiled artifacts in"`
	Args         struct {
		Packages []string `positional-arg-name:"packages" description:"The packages to compile"`
	} `positional-args:"true" required:"true"`
}{
	Usage: `
please-go-install is shipped with Please and is used to build go modules similarly to go install. 

Unlike 'go install', this tool doesn't rely on the go path or modules to find its dependencies. Instead it takes in 
go import config just like 'go tool compile/link -importcfg'. 

This tool determines the dependencies between packages and output a commands in the correct order to compile them. 

`,
}

type pkgGraph struct {
	pkgs map[string]bool
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		panic(err)
	}

	tc := &toolchain.Toolchain{
		CcTool: opts.CCTool,
		GoTool: opts.GoTool,
		Exec:   &exec.OsExecutor{Stdout: os.Stdout, Stderr: os.Stderr},
	}

	initBuildEnv(tc)
	pkgs := parseImportConfig()

	for _, target := range opts.Args.Packages {
		if !strings.HasPrefix(target, opts.ModuleName) {
			target = filepath.Join(opts.ModuleName, target)
		}
		if strings.HasSuffix(target, "/...") {
			importRoot := strings.TrimSuffix(target, "/...")
			pkgRoot := pkgDir(importRoot)
			err := filepath.Walk(pkgRoot, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					panic(err)
				}
				if !info.IsDir() {
					relativePackage := filepath.Dir(strings.TrimPrefix(path, pkgRoot))
					if err := pkgs.compile(tc, []string{}, filepath.Join(importRoot, relativePackage)); err != nil {
						switch err.(type) {
						case *build.NoGoError:
							// We might walk into a dir that has no .go files for the current arch. This shouldn't
							// be an error so we just eat this
							return nil
						default:
							return err
						}
					}
				}
				return nil
			})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			if err := pkgs.compile(tc, []string{}, target); err != nil {
				log.Fatalf("Failed to compile %v: %v", target, err)
			}
		}
	}
}

func initBuildEnv(tc *toolchain.Toolchain) {
	tc.Exec.Exec("mkdir -p %s\n", filepath.Join(opts.Out, "bin"))
	tc.Exec.Exec("touch %s", opts.LDFlags)
}

// pkgDir returns the file path to the given target package
func pkgDir(target string) string {
	p := strings.TrimPrefix(target, opts.ModuleName)
	return filepath.Join(opts.SrcRoot, p)
}

func parseImportConfig() *pkgGraph {
	pkgs := &pkgGraph{
		pkgs: map[string]bool{
			"unsafe": true, // Not sure how many other packages like this I need to handle
			"C":      true, // Pseudo-package for cgo symbols
		},
	}

	if opts.ImportConfig != "" {
		f, err := os.Open(opts.ImportConfig)
		if err != nil {
			panic(fmt.Sprint("failed to open import config: " + err.Error()))
		}
		defer f.Close()

		importCfg := bufio.NewScanner(f)
		for importCfg.Scan() {
			line := importCfg.Text()
			parts := strings.Split(strings.TrimPrefix(line, "packagefile "), "=")
			pkgs.pkgs[parts[0]] = true
		}
	}
	return pkgs
}

func checkCycle(path []string, next string) ([]string, error) {
	for i, p := range path {
		if p == next {
			return nil, fmt.Errorf("package cycle detected: \n%s", strings.Join(append(path[i:], next), "\n ->"))
		}
	}

	return append(path, next), nil
}

func (g *pkgGraph) compile(tc *toolchain.Toolchain, from []string, target string) error {
	if done := g.pkgs[target]; done {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Compiling package %s from %v\n", target, from)

	from, err := checkCycle(from, target)
	if err != nil {
		return err
	}

	pkgDir := pkgDir(target)
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
		err := g.compile(tc, from, i)
		if err != nil {
			if strings.Contains(err.Error(), "cannot find package") {
				// Go will fail to find this import and provide a much better message than we can
				continue
			}
			return err
		}
	}

	err = compilePackage(tc, target, pkg)
	if err != nil {
		return err
	}
	g.pkgs[target] = true
	return nil
}

func compilePackage(tc *toolchain.Toolchain, target string, pkg *build.Package) error {
	allSrcs := append(append(pkg.CFiles, pkg.GoFiles...), pkg.HFiles...)
	if len(pkg.GoFiles)+len(pkg.CgoFiles) == 0 {
		return nil
	}

	out := fmt.Sprintf("%s/%s.a", opts.Out, target)
	workDir := fmt.Sprintf("_build/%s", target)

	tc.Exec.Exec("mkdir -p %s", workDir)
	tc.Exec.Exec("mkdir -p %s", filepath.Dir(out))
	tc.Exec.Exec("ln %s %s", toolchain.FullPaths(allSrcs, pkg.Dir), workDir)

	goFiles := pkg.GoFiles

	var objFiles []string

	if len(pkg.CgoFiles) > 0 {
		cFiles := pkg.CFiles

		cgoGoFiles, cgoCFiles := tc.CGO(pkg.Dir, workDir, pkg.CgoFiles)
		goFiles = append(goFiles, cgoGoFiles...)
		cFiles = append(cFiles, cgoCFiles...)

		cObjFiles := tc.CCompile(workDir, cFiles, pkg.CgoCFLAGS)
		objFiles = append(objFiles, cObjFiles...)
	}

	if len(pkg.SFiles) > 0 {
		asmH, symabis := tc.Symabis(pkg.Dir, workDir, pkg.SFiles)

		tc.GoAsmCompile(workDir, opts.ImportConfig, out, goFiles, asmH, symabis)

		asmObjFiles := tc.Asm(pkg.Dir, workDir, pkg.SFiles)
		objFiles = append(objFiles, asmObjFiles...)
	} else {
		tc.GoCompile(workDir, opts.ImportConfig, out, goFiles)
	}

	if len(objFiles) > 0 {
		tc.Pack(workDir, out, objFiles)
	}

	tc.Exec.Exec("echo \"packagefile %s=%s\" >> %s", target, out, opts.ImportConfig)
	if len(pkg.CgoLDFLAGS) > 0 {
		tc.Exec.Exec("echo -n \"%s\" >> %s", strings.Join(pkg.CgoLDFLAGS, " "), opts.LDFlags)
	}

	if pkg.IsCommand() {
		filename := strings.TrimSuffix(filepath.Base(out), ".a")
		binName := filepath.Join(opts.Out, "bin", filename)

		tc.Link(out, binName, opts.ImportConfig, opts.LDFlags)
	}
	return nil
}
