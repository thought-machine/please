// Package main implements please_pex, which builds Python pex files for us.
package main

import (
	"gopkg.in/op/go-logging.v1"

	"cli"
	"tools/please_pex/pex"
)

var log = logging.MustGetLogger("please_pex")

var opts = struct {
	Usage       string
	Verbosity   int      `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Out         string   `short:"o" long:"out" env:"OUT" description:"Output file"`
	EntryPoint  string   `short:"e" long:"entry_point" env:"SRC" description:"Entry point to pex file"`
	ModuleDir   string   `short:"m" long:"module_dir" description:"Python module dir to implicitly load modules from"`
	TestSrcs    []string `long:"test_srcs" env:"SRCS" env-delim:" " description:"Test source files"`
	Test        bool     `short:"t" long:"test" description:"True if we're to build a test"`
	Interpreter string   `short:"i" long:"interpreter" env:"TOOLS_INTERPRETER" description:"Python interpreter to use"`
	Shebang     string   `short:"s" long:"shebang" description:"Explicitly set shebang to this"`
	ZipSafe     bool     `long:"zip_safe" description:"Marks this pex as zip-safe"`
	NoZipSafe   bool     `long:"nozip_safe" description:"Marks this pex as zip-unsafe"`
	CodeHash    string   `long:"code_hash" env:"STAMP" description:"Hash to embed into the pex file"`
}{
	Usage: `
please_pex is a tool to create .pex files for Python.

Pex files are the binaries that Please outputs for its python_binary
and python_test rules. They are essentially self-contained executable
Python environments.

See https://github.com/pantsbuild/pex for more info about pex.
`,
}

func main() {
	cli.ParseFlagsOrDie("please_pex", "8.1.2", &opts)
	cli.InitLogging(opts.Verbosity)
	w := pex.NewPexWriter(opts.EntryPoint, opts.Interpreter, opts.CodeHash, !opts.NoZipSafe)
	if opts.Shebang != "" {
		w.SetShebang(opts.Shebang)
	}
	if opts.Test {
		w.SetTest(opts.TestSrcs)
	}
	if err := w.Write(opts.Out, opts.ModuleDir); err != nil {
		log.Fatalf("%s", err)
	}
}
