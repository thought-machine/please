// Package main implements a standalone parser binary,
// which is simply a benchmark for how fast we can read a large number
// of BUILD files.
package main

import (
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/parse/rules"
)

var log = logging.MustGetLogger("parser")

var opts = struct {
	Usage        string
	Verbosity    cli.Verbosity `short:"v" long:"verbosity" default:"notice" description:"Verbosity of output (higher number = more output)"`
	NumThreads   int           `short:"n" long:"num_threads" default:"10" description:"Number of concurrent parse threads to run"`
	ParseOnly    bool          `short:"p" long:"parse_only" description:"Only parse input files, do not interpret them."`
	DumpAst      bool          `short:"d" long:"dump_ast" description:"Prints AST to stdout. Implies --parse_only."`
	NoConfig     bool          `long:"no_config" description:"Don't look for or load a .plzconfig file"`
	BuildDefsDir string        `short:"b" long:"build_defs_dir" description:"Load build_defs files from this directory. This assumes that they are all produced by trivial build rules with obvious names. They will need to be built first."`
	Args         struct {
		BuildFiles []string `positional-arg-name:"files" required:"true" description:"BUILD files to parse"`
	} `positional-args:"true"`
}{
	Usage: `Test parser for BUILD files using our standalone parser.`,
}

func parseFile(pkg *core.Package, p *asp.Parser, filename string) error {
	if opts.ParseOnly || opts.DumpAst {
		stmts, err := p.ParseFileOnly(filename)
		if opts.DumpAst {
			config := spew.NewDefaultConfig()
			config.DisablePointerAddresses = true
			config.DisableLengths = true
			config.DisableTypes = true
			config.OmitEmpty = true
			config.Indent = "  "
			os.Stdout.Write([]byte(cleanup(config.Sdump(stmts))))
		}
		return err
	}
	return p.ParseFile(pkg, filename)
}

// cleanup runs a few arbitrary cleanup steps on the given AST dump.
// We do our best to do it analytically but one or two parts are a bit hard to alter.
func cleanup(ast string) string {
	r := regexp.MustCompile(`\n *Pos: .*\n`)
	ast = r.ReplaceAllString(ast, "\n")
	r = regexp.MustCompile(`String: "\\"(.*)\\"",`)
	return r.ReplaceAllString(ast, `String: "$1",`)
}

func mustLoadBuildDefsDir(state *core.BuildState, dirname string) {
	dir, err := ioutil.ReadDir(dirname)
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
		config, err = core.ReadConfigFiles([]string{
			path.Join(core.MustFindRepoRoot(), core.ConfigFileName),
		}, "")
		if err != nil {
			log.Fatalf("%s", err)
		}
	}

	state := core.NewBuildState(opts.NumThreads, nil, int(opts.Verbosity), config)
	if opts.BuildDefsDir != "" {
		mustLoadBuildDefsDir(state, opts.BuildDefsDir)
	}

	ch := make(chan string, 100)
	var wg sync.WaitGroup
	wg.Add(opts.NumThreads)
	total := len(opts.Args.BuildFiles)
	p := asp.NewParser(state)

	log.Debug("Loading built-in build rules...")
	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	for _, filename := range dir {
		if strings.HasSuffix(filename, ".gob") {
			srcFile := strings.TrimSuffix(filename, ".gob")
			src, _ := rules.Asset(srcFile)
			p.MustLoadBuiltins("src/parse/rules/"+srcFile, src, rules.MustAsset(filename))
		}
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
