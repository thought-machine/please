// Package main implements a standalone parser binary,
// which is simply a benchmark for how fast we can read a large number
// of BUILD files.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

var log = logging.Log

var opts = struct {
	Usage        string
	Verbosity    cli.Verbosity `short:"v" long:"verbosity" default:"notice" description:"Verbosity of output (higher number = more output)"`
	NumThreads   int           `short:"n" long:"num_threads" default:"10" description:"Number of concurrent parse threads to run"`
	ParseOnly    bool          `short:"p" long:"parse_only" description:"Only parse input files, do not interpret them."`
	DumpAst      bool          `short:"d" long:"dump_ast" description:"Prints AST to stdout. Implies --parse_only."`
	Check        bool          `short:"c" long:"check" description:"Runs some static checks on the parsed files. Implies --parse_only."`
	NoConfig     bool          `long:"no_config" description:"Don't look for or load a .plzconfig file"`
	BuildDefsDir string        `short:"b" long:"build_defs_dir" description:"Load build_defs files from this directory. This assumes that they are all produced by trivial build rules with obvious names. They will need to be built first."`
	Args         struct {
		BuildFiles []string `positional-arg-name:"files" required:"true" description:"BUILD files to parse"`
	} `positional-args:"true"`
}{
	Usage: `Test parser for BUILD files using our standalone parser.`,
}

func parseFile(pkg *core.Package, p *asp.Parser, filename string) error {
	if opts.ParseOnly || opts.DumpAst || opts.Check {
		stmts, err := p.ParseFileOnly(filename)
		if opts.Check && err == nil {
			if errs := checkAST(stmts); len(errs) != 0 {
				for _, err := range errs {
					printErr(err)
				}
				return fmt.Errorf("Errors found while checking %s", filename)
			}
		}
		if opts.DumpAst {
			config := spew.NewDefaultConfig()
			config.DisablePointerAddresses = true
			config.Indent = "  "
			os.Stdout.Write([]byte(cleanup(config.Sdump(stmts))))
		}
		return err
	}
	return p.ParseFile(pkg, nil, nil, false, filename)
}

type assignment struct {
	Name string
	Pos  asp.Position
	Read bool // does it get read later on?
}

// checkAST runs some static checks on a loaded AST.
// Currently this checks for variables that are assigned to but not read.
func checkAST(stmts []*asp.Statement, parentScopes ...map[string]assignment) (errs []assignment) {
	assigns := map[string]assignment{}
	allScopes := append(parentScopes, assigns)

	markAssign := func(name string) {
		// Loop backward through scopes so we're doing it in correct order
		for i := len(allScopes) - 1; i >= 0; i-- {
			if assign, present := allScopes[i][name]; present {
				allScopes[i][name] = assignment{Name: name, Pos: assign.Pos, Read: true}
			}
		}
	}

	asp.WalkAST(stmts, func(ident *asp.IdentStatement) bool {
		if ident.Action != nil && ident.Action.Assign != nil {
			if _, present := assigns[ident.Name]; !present {
				assigns[ident.Name] = assignment{Name: ident.Name, Pos: ident.Action.Assign.Pos}
			}
		}
		return true
	}, func(def *asp.FuncDef) bool {
		return false // do nothing for now, we'll handle it for real below
	}, func(ident *asp.IdentExpr) bool {
		markAssign(ident.Name)
		return true
	}, func(v *asp.FStringVar) bool {
		if len(v.Var) == 1 {
			markAssign(v.Var[0])
		}
		return false // never anything interesting from here
	})
	// Do it again to recurse into nested functions (the ordering here is important for functions that
	// are defined before the variables they read)
	asp.WalkAST(stmts, func(def *asp.FuncDef) bool {
		errs = append(errs, checkAST(def.Statements, allScopes...)...)
		return false
	})
	for _, assign := range assigns {
		if !assign.Read {
			errs = append(errs, assign)
		}
	}
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].Pos.Line < errs[j].Pos.Line
	})
	return errs
}

func printErr(err assignment) {
	stack := asp.AddStackFrame(err.Pos, fmt.Errorf("Variable %s is written but never read", err.Name))
	if f, err := os.Open(err.Pos.Filename); err == nil {
		defer f.Close()
		stack = asp.AddReader(stack, f)
	}
	fmt.Printf("%s\n", stack)
}

// cleanup runs a few arbitrary cleanup steps on the given AST dump.
// We do our best to do it analytically but one or two parts are a bit hard to alter.
func cleanup(ast string) string {
	r := regexp.MustCompile(`\n *Pos: .*\n`)
	ast = r.ReplaceAllString(ast, "\n")
	r = regexp.MustCompile(`String: "\\"(.*)\\"",`)
	ast = r.ReplaceAllString(ast, `String: "$1",`)
	r = regexp.MustCompile(`: \(len=[0-9]+\) "`)
	return r.ReplaceAllString(ast, `: "`)
}

func mustLoadBuildDefsDir(state *core.BuildState, dirname string) {
	dir, err := os.ReadDir(dirname)
	if err != nil {
		log.Fatalf("%s", err)
	}
	for _, fi := range dir {
		if strings.HasSuffix(fi.Name(), ".build_defs") {
			t := core.NewBuildTarget(core.NewBuildLabel(dirname, strings.TrimSuffix(fi.Name(), ".build_defs")))
			t.AddOutput(fi.Name())
			t.SetState(core.Built)
			state.Graph.AddTarget(t)
		}
	}
}

func main() {
	cli.ParseFlagsOrDie("parser", &opts)
	cli.InitLogging(opts.Verbosity)

	config := core.DefaultConfiguration()
	if !opts.NoConfig {
		var err error
		config, err = core.ReadConfigFiles([]string{filepath.Join(core.MustFindRepoRoot(), core.ConfigFileName)}, nil)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}
	config.Please.NumThreads = opts.NumThreads

	state := core.NewBuildState(config)
	if opts.BuildDefsDir != "" {
		mustLoadBuildDefsDir(state, opts.BuildDefsDir)
	}

	ch := make(chan string, 100)
	var wg sync.WaitGroup
	wg.Add(opts.NumThreads)
	total := len(opts.Args.BuildFiles)
	p := asp.NewParser(state)

	log.Debug("Loading built-in build rules...")
	dir, _ := rules.AllAssets(state.ExcludedBuiltinRules())
	sort.Strings(dir)
	for _, filename := range dir {
		src, _ := rules.ReadAsset(filename)
		p.MustLoadBuiltins(filename, src)
	}

	start := time.Now()
	var errors int64
	for i := 0; i < opts.NumThreads; i++ {
		go func() {
			for file := range ch {
				pkg := core.NewPackage(file)
				pkg.Filename = file
				if err := parseFile(pkg, p, file); err != nil {
					atomic.AddInt64(&errors, 1)
					log.Error("Error parsing %s: %s", file, err)
				}
			}
			wg.Done()
		}()
	}

	for _, file := range opts.Args.BuildFiles {
		ch <- file
	}
	close(ch)
	wg.Wait()

	log.Notice("Parsed %d files in %s", total, time.Since(start))
	log.Notice("Success: %d / %d (%0.2f%%)", total-int(errors), total, 100.0*float64(total-int(errors))/float64(total))
}
