package main

import (
	"bufio"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/go-flags"
)

var opts = struct {
	Usage string

	SrcRoot      string `short:"r" long:"src_root" description:"The src root of the module to inspect" default:"."`
	ModuleName   string `short:"n" long:"module_name" description:"The name of the module"`
	ImportConfig string `short:"i" long:"importcfg" description:"the import config for the modules dependencies"`
	GoTool       string `short:"g" long:"go_tool" description:"The location of the go binary"`
	CCTool       string `short:"c" long:"cc_tool" description:"The c compiler to use"`
	Out          string `short:"o" long:"out" description:"The output directory to put compiled artifacts in"`
	Args         struct {
		Packages []string `positional-arg-name:"packages" description:"The packages to compile"`
	} `positional-args:"true" required:"true"`
	GOROOT string `env:"GOROOT"`
	GOOS   string `env:"GOOS"`
	GOARCH string `env:"GOARCH"`
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

		importCfg := bufio.NewScanner(f)
		for importCfg.Scan() {
			line := importCfg.Text()
			parts := strings.Split(strings.TrimPrefix(line, "packagefile "), "=")
			pkgs.pkgs[parts[0]] = true
		}
	}
	fmt.Println("#!/bin/sh")
	fmt.Println("set -e")
	for _, target := range opts.Args.Packages {
		if strings.HasSuffix(target, "/...") {
			root := strings.TrimSuffix(target, "/...")
			err := filepath.Walk(filepath.Join(opts.SrcRoot, root), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					panic(err)
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
					pkgs.compile([]string{}, strings.TrimPrefix(filepath.Dir(path), opts.SrcRoot))
				}
				return nil
			})
			if err != nil {
				panic(err)
			}
		} else {
			pkgs.compile([]string{}, target)
		}
	}
}

func checkCycle(path []string, next string) []string {
	for i, p := range path {
		if p == next {
			panic(fmt.Sprintf("Package cycle detected: \n%s", strings.Join(append(path[i:], next), "\n ->")))
		}
	}

	return append(path, next)
}

func (g *pkgGraph) compile(from []string, target string) {
	if done := g.pkgs[target]; done {
		return
	}

	from = checkCycle(from, target)

	pkgDir := filepath.Join(opts.SrcRoot, target)
	// The package name can differ from the directory it lives in, in which case the parent directory is the one we want
	if _, err := os.Lstat(pkgDir); os.IsNotExist(err) {
		pkgDir = filepath.Dir(pkgDir)
	}

	// TODO(jpoole): is import vendor the correct thing to do here?
	pkg, err := build.ImportDir(pkgDir, build.ImportComment)
	if err != nil {
		panic(err)
	}

	for _, i := range pkg.Imports {
		g.compile(from, i)
	}

	compilePackage(target, pkg)
	g.pkgs[target] = true
}

func compilePackage(target string, pkg *build.Package) {
	tc := &toolchain{
		ccTool: opts.CCTool,
		goTool: opts.GoTool,
	}
	allSrcs := append(append(pkg.CFiles, pkg.GoFiles...), pkg.HFiles...)

	out := fmt.Sprintf("%s/%s.a", opts.Out, target)
	workDir := fmt.Sprintf("_build/%s", target)

	fmt.Printf("mkdir -p %s\n", workDir)
	fmt.Printf("mkdir -p %s\n", filepath.Dir(out))
	fmt.Printf("ln %s %s\n", fullPaths(allSrcs, pkg.Dir), workDir)

	goFiles := pkg.GoFiles

	var objFiles []string

	if len(pkg.CgoFiles) > 0 {
		cFiles := pkg.CFiles

		cgoGoFiles, cgoCFiles := tc.cgo(pkg.Dir, workDir, pkg.CgoFiles)
		goFiles = append(goFiles, cgoGoFiles...)
		cFiles = append(cFiles, cgoCFiles...)

		cObjFiles := tc.cCompile(workDir, cFiles, pkg.CgoCFLAGS)
		objFiles = append(objFiles, cObjFiles...)
	}

	if len(pkg.SFiles) > 0 {
		asmH, symabis := tc.symabis(pkg.Dir, workDir, target, pkg.SFiles)

		tc.goAsmCompile(workDir, opts.ImportConfig, out, goFiles, asmH, symabis)

		asmObjFiles := tc.asm(pkg.Dir, workDir, target, pkg.SFiles)
		objFiles = append(objFiles, asmObjFiles...)
	} else {
		tc.goCompile(workDir, opts.ImportConfig, out, goFiles)
	}

	if len(objFiles) > 0 {
		tc.pack(workDir, out, objFiles)
	}

	fmt.Printf("echo \"packagefile %s=%s\" >> %s\n", target, out, opts.ImportConfig)

	if pkg.IsCommand() {
		filename := strings.TrimSuffix(filepath.Base(out), ".a")
		binName := filepath.Join(opts.Out, "bin", filename)

		// TODO(jpoole): This could probably be done in some sort of init phase
		fmt.Printf("mkdir -p %s\n", filepath.Dir(binName))
		tc.link(out, binName)
	}
}
