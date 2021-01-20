package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var opts = struct {
	Usage string

	SrcRoot      string   `short:"r" long:"src_root" description:"The src root of the module to inspect" default:"."`
	ModuleName   string   `short:"n" long:"module_name" description:"The name of the module"`
	ImportConfig string   `short:"i" long:"importcfg" description:"the import config for the modules dependencies"`
	GoTool       string   `short:"g" long:"go" description:"The location of the go binary"`
	CCTool       string   `short:"g" long:"go" description:"The c compiler to use"`
	Packages     []string `short:"p" long:"packages" description:"The target packages to list dependencies for" default:"."`
	Out          string
}{
	Usage: `
please-go-install is shipped with Please and is used to build go modules similarly to go install. 

Unlike 'go install', this tool doesn't rely on the go path or modules to find its dependencies. Instead it takes in 
go import config just like 'go tool compile/link -importcfg'. 

This tool determines the dependencies between packages and output a commands in the correct order to compile them. 

`,
	SrcRoot:      os.Args[1],
	ModuleName:   os.Args[2],
	ImportConfig: os.Args[3],
	GoTool:       os.Args[4],
	CCTool:       os.Args[5],
	Out:          os.Args[6],
	Packages:     os.Args[7:],
}

var fileSet = token.NewFileSet()

type pkgGraph struct {
	pkgs map[string]bool
}

func main() {
	pkgs := &pkgGraph{
		pkgs: map[string]bool{
			"unsafe": true, // Not sure how many other packages like this I need to handle
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
	for _, target := range opts.Packages {
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

	packageASTs, err := parser.ParseDir(fileSet, pkgDir, nil, 0)
	if err != nil {
		panic(err)
	}

	var targetPackage *ast.Package
	for k, v := range packageASTs {
		if !strings.HasSuffix(k, "_test") {
			targetPackage = v
			break
		}
	}
	if targetPackage == nil {
		panic(fmt.Sprintf("couldn't find package %s in %s: %v", target, pkgDir, packageASTs))
	}

	var srcs []string
	var cgoSrcs []string
	anyCGO := false
	for name, f := range targetPackage.Files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}

		matched, err := build.Default.MatchFile(filepath.Dir(name), filepath.Base(name))
		if err != nil {
			panic(err)
		}
		if !matched {
			continue
		}
		cgo := false
		for _, i := range f.Imports {
			name := strings.TrimSuffix(strings.TrimPrefix(i.Path.Value, "\""), "\"")
			if name == "C" {
				anyCGO = true
				cgo = true
			} else {
				g.compile(from, name)
			}
		}

		if cgo {
			cgoSrcs = append(cgoSrcs, name)
		} else {
			srcs = append(srcs, name)
		}
	}
	binary := targetPackage.Name == "main"
	if anyCGO {
		goToolCGOCompile(target, binary, pkgDir, srcs, cgoSrcs)
	} else {
		goToolCompile(target, binary, srcs) // output the package as ready to be compiled
	}
	g.pkgs[target] = true
}

func goToolCompile(target string, binary bool, srcs []string) {
	out := fmt.Sprintf("%s/%s.a", opts.Out, target)

	prepOutDir := fmt.Sprintf("mkdir -p %s", filepath.Dir(out))
	compile := fmt.Sprintf("%s tool compile -pack -complete -importcfg %s -o %s %s", opts.GoTool, opts.ImportConfig, out, strings.Join(srcs, " "))
	updateImportCfg := fmt.Sprintf("echo \"packagefile %s=%s\" >> %s", target, out, opts.ImportConfig)

	fmt.Println(prepOutDir)
	fmt.Println(compile)
	fmt.Println(updateImportCfg)

	if binary {
		goToolLink(out)
	}
}

func goToolCGOCompile(target string, binary bool, pkgDir string, srcs []string, cgoSrcs []string) {
	out := fmt.Sprintf("%s/%s.a", opts.Out, target)

	// We need to operate out of the package working directory for the cpp compiler to play ball so trim the package dir
	// from the source paths.
	for i := range srcs {
		srcs[i] = strings.TrimPrefix(strings.TrimPrefix(srcs[i], pkgDir), "/")
	}

	for i := range cgoSrcs {
		cgoSrcs[i] = strings.TrimPrefix(strings.TrimPrefix(cgoSrcs[i], pkgDir), "/")
	}

	prepOutDir := fmt.Sprintf("mkdir -p %s", filepath.Dir(out))
	cdPkgDir := fmt.Sprintf("cd %s", pkgDir)
	generateCGO := fmt.Sprintf("%s tool cgo %s", opts.GoTool, strings.Join(cgoSrcs, " "))
	compileGo := fmt.Sprintf("%s tool compile -pack -importcfg $OLDPWD/%s -o out.a _obj/*.go", opts.GoTool, opts.ImportConfig)
	compileCGO := fmt.Sprintf("%s -Wno-error -ldl -Wno-unused-parameter -c -I _obj -I . _obj/_cgo_export.c _obj/*.cgo2.c", opts.CCTool)
	compileC := fmt.Sprintf("%s -Wno-error -ldl -Wno-unused-parameter -c -I _obj -I . *.c", opts.CCTool)
	mergeArchive := fmt.Sprintf("%s tool pack r out.a *.o ", opts.GoTool)
	moveArchive := fmt.Sprintf("cd $OLDPWD && mv %s/out.a %s", pkgDir, out)
	updateImportCfg := fmt.Sprintf("echo \"packagefile %s=%s\" >> %s", target, out, opts.ImportConfig)


	fmt.Println(prepOutDir)
	fmt.Println(cdPkgDir)
	fmt.Println(generateCGO)

	// Copy non-cgo srcs to _obj dir
	for _, src := range srcs {
		fmt.Printf("ln %s _obj/\n", src)
	}

	fmt.Println(compileGo)
	fmt.Println(compileC)
	fmt.Println(compileCGO)
	fmt.Println(mergeArchive)
	fmt.Println(moveArchive)
	fmt.Println(updateImportCfg)

	if binary {
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
