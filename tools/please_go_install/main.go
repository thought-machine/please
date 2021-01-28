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

	if len(pkg.CgoFiles) > 0 {
		goToolCGOCompile(target, pkg)
	} else {
		goToolCompile(target, pkg) // output the package as ready to be compiled
	}
	g.pkgs[target] = true
}

func goToolCompile(target string, pkg *build.Package) {
	fullSrcPaths := make([]string, len(pkg.GoFiles))
	for i, src := range pkg.GoFiles {
		fullSrcPaths[i] = filepath.Join(pkg.Dir, src)
	}
	out := fmt.Sprintf("%s/%s.a", opts.Out, target)

	prepOutDir := fmt.Sprintf("mkdir -p %s", filepath.Dir(out))
	compile := fmt.Sprintf("%s tool compile -pack -complete -importcfg %s -o %s %s", opts.GoTool, opts.ImportConfig, out, strings.Join(fullSrcPaths, " "))
	updateImportCfg := fmt.Sprintf("echo \"packagefile %s=%s\" >> %s", target, out, opts.ImportConfig)

	fmt.Println(prepOutDir)
	fmt.Println(compile)
	fmt.Println(updateImportCfg)

	if pkg.IsCommand() {
		goToolLink(out)
	}
}

func goToolCGOCompile(target string, pkg *build.Package) {
	out := fmt.Sprintf("%s/%s.a", opts.Out, target)

	prepOutDir := fmt.Sprintf("mkdir -p %s", filepath.Dir(out))
	cdPkgDir := fmt.Sprintf("cd %s", pkg.Dir)
	generateCGO := fmt.Sprintf("%s tool cgo %s", opts.GoTool, strings.Join(pkg.CgoFiles, " "))
	compileGo := fmt.Sprintf("%s tool compile -pack -importcfg $OLDPWD/%s -o out.a _obj/*.go", opts.GoTool, opts.ImportConfig)
	// TODO(jpoole): We can use pkg to determine the correct cgo flags to pass to the compiler here
	compileCGO := fmt.Sprintf("%s -Wno-error -ldl -Wno-unused-parameter -c -I _obj -I . _obj/_cgo_export.c _obj/*.cgo2.c", opts.CCTool)
	compileC := fmt.Sprintf("%s -Wno-error -ldl -Wno-unused-parameter -c -I _obj -I . *.c", opts.CCTool)
	mergeArchive := fmt.Sprintf("%s tool pack r out.a *.o ", opts.GoTool)
	moveArchive := fmt.Sprintf("cd $OLDPWD && mv %s/out.a %s", pkg.Dir, out) //TODO(jpoole): can we just output this in the right place to begin with?
	updateImportCfg := fmt.Sprintf("echo \"packagefile %s=%s\" >> %s", target, out, opts.ImportConfig)

	fmt.Println(prepOutDir)
	fmt.Println(cdPkgDir)
	fmt.Println(generateCGO)

	// TODO(jpoole): we should probably create our own work dir rather than creating _obj in the pkg dir
	// Copy non-cgo srcs to _obj dir
	for _, src := range pkg.GoFiles {
		fmt.Printf("ln %s _obj/\n", src)
	}

	fmt.Println(compileGo)
	fmt.Println(compileC)
	fmt.Println(compileCGO)
	fmt.Println(mergeArchive)
	fmt.Println(moveArchive)
	fmt.Println(updateImportCfg)

	if pkg.IsCommand() {
		goToolLink(out)
	}
}

func goToolLink(archive string) {
	filename := strings.TrimSuffix(filepath.Base(archive), ".a")
	out := filepath.Join(opts.Out, "bin", filename)
	link := fmt.Sprintf("%s tool link -importcfg %s -o %s %s", opts.GoTool, opts.ImportConfig, out, archive)
	fmt.Printf("mkdir -p %s\n", filepath.Dir(out))
	fmt.Println(link)
}
