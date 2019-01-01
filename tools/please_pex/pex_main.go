// Package main implements please_pex, which builds runnable Python zip files for us.
package main

import (
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/tools/please_pex/pex"
)

var log = logging.MustGetLogger("please_pex")

var opts = struct {
	Usage       string
	Verbosity   cli.Verbosity `short:"v" long:"verbosity" default:"warning" description:"Verbosity of output (higher number = more output)"`
	Out         string        `short:"o" long:"out" env:"OUT" description:"Output file"`
	EntryPoint  string        `short:"e" long:"entry_point" env:"SRC" description:"Entry point to pex file"`
	ModuleDir   string        `short:"m" long:"module_dir" description:"Python module dir to implicitly load modules from"`
	TestSrcs    []string      `long:"test_srcs" env:"SRCS" env-delim:" " description:"Test source files"`
	Test        bool          `short:"t" long:"test" description:"True if we're to build a test"`
	Interpreter string        `short:"i" long:"interpreter" env:"TOOLS_INTERPRETER" description:"Python interpreter to use"`
	TestRunner  string        `short:"r" long:"test_runner" choice:"unittest" choice:"pytest" choice:"behave" default:"unittest" description:"Test runner to use"`
	Shebang     string        `short:"s" long:"shebang" description:"Explicitly set shebang to this"`
	ZipSafe     bool          `long:"zip_safe" description:"Marks this pex as zip-safe"`
	NoZipSafe   bool          `long:"nozip_safe" description:"Marks this pex as zip-unsafe"`
}{
	Usage: `
please_pex is a tool to create .pex files for Python.

These are not really pex files any more, they are just zip files (which Python supports
out of the box). They still have essentially the same approach of containing all the
dependent code as a self-contained self-executable environment.
`,
}

func main() {
	cli.ParseFlagsOrDie("please_pex", &opts)
	cli.InitLogging(opts.Verbosity)
	w := pex.NewWriter(opts.EntryPoint, opts.Interpreter, !opts.NoZipSafe)
	if opts.Shebang != "" {
		w.SetShebang(opts.Shebang)
	}
	if opts.Test {
		w.SetTest(opts.TestSrcs, opts.TestRunner)
	}
	if err := w.Write(opts.Out, opts.ModuleDir); err != nil {
		log.Fatalf("%s", err)
	}
}
