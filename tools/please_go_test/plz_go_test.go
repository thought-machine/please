// Package main implements plz_go_test, a code templater for go tests.
// This is essentially equivalent to what 'go test' does but it lifts some restrictions
// on file organisation and allows the code to be instrumented for coverage as separate
// build targets rather than having to repeat it for every test.
package main

import (
	"os"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/tools/please_go_test/gotest"
)

var log = logging.MustGetLogger("plz_go_test")

var opts struct {
	Usage      string        `usage:"please_go_test is a code templater for Go tests.\n\nIt writes out the test main file required for each test, similar to what 'go test' does but as a separate tool that Please can invoke."`
	Verbosity  cli.Verbosity `short:"v" long:"verbosity" default:"warning" description:"Verbosity of output (higher number = more output)"`
	Dir        string        `short:"d" long:"dir" description:"Directory to search for Go package files for coverage"`
	Exclude    []string      `short:"x" long:"exclude" default:"third_party/go" description:"Directories to exclude from search"`
	Output     string        `short:"o" long:"output" description:"Output filename" required:"true"`
	Package    string        `short:"p" long:"package" description:"Package containing this test" env:"PKG"`
	ImportPath string        `short:"i" long:"import_path" description:"Full import path to the package"`
	Args       struct {
		Go      string   `positional-arg-name:"go" description:"Location of go command" required:"true"`
		Sources []string `positional-arg-name:"sources" description:"Test source files" required:"true"`
	} `positional-args:"true" required:"true"`
}

func main() {
	cli.ParseFlagsOrDie("plz_go_test", &opts)
	cli.InitLogging(opts.Verbosity)
	coverVars, err := gotest.FindCoverVars(opts.Dir, opts.ImportPath, opts.Exclude, opts.Args.Sources)
	if err != nil {
		log.Fatalf("Error scanning for coverage: %s", err)
	}
	if err = gotest.WriteTestMain(opts.Package, opts.ImportPath, gotest.IsVersion18(opts.Args.Go), opts.Args.Sources, opts.Output, coverVars); err != nil {
		log.Fatalf("Error writing test main: %s", err)
	}
	os.Exit(0)
}
