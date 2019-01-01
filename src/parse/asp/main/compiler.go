// Package main implements a compiler for the builtin build rules, which is used at bootstrap time.
package main

import (
	"os"
	"path"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

var log = logging.MustGetLogger("asp")

var opts = struct {
	Usage     string
	Verbosity cli.Verbosity `short:"v" long:"verbosity" default:"notice" description:"Verbosity of output (higher number = more output)"`
	OutputDir string        `short:"o" long:"output_dir" required:"true" description:"Output directory"`
	Args      struct {
		BuildFiles []string `positional-arg-name:"files" required:"true" description:"BUILD files to parse"`
	} `positional-args:"true"`
}{
	Usage: `Compiler for built-in build rules.`,
}

func main() {
	cli.ParseFlagsOrDie("parser", &opts)
	cli.InitLogging(opts.Verbosity)

	if err := os.MkdirAll(opts.OutputDir, os.ModeDir|0775); err != nil {
		log.Fatalf("%s", err)
	}
	p := asp.NewParser(core.NewDefaultBuildState())
	for _, filename := range opts.Args.BuildFiles {
		out := path.Join(opts.OutputDir, path.Base(filename)) + ".gob"
		if err := p.ParseToFile(filename, out); err != nil {
			log.Fatalf("%s", err)
		}
	}
}
