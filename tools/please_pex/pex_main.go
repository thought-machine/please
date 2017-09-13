// Package main implements please_pex, which builds Python pex files for us.
package main

import (
	"crypto/sha1"
	"encoding/base64"

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
	CodeHash    string   `long:"code_hash" description:"Hash to embed into the pex file"`
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
	cli.ParseFlagsOrDie("please_pex", "9.0.1", &opts)
	cli.InitLogging(opts.Verbosity)
	if opts.CodeHash == "" {
		// It is difficult to know what the code hash should be. We want to have one to keep
		// pex happy, but it is not "correct" since we don't have the entire pex to write at this
		// point. It doesn't really matter in the sense that when run within plz pex will never
		// reuse its code hashes anyway (because $HOME is overridden).
		sum := sha1.Sum([]byte(opts.Out))
		opts.CodeHash = base64.RawURLEncoding.EncodeToString(sum[:])
	}
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
