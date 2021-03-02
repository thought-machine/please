package main

import (
	"github.com/thought-machine/please/tools/please_go/test"
	"log"
	"os"

	"github.com/peterebden/go-cli-init/v4/flags"

	"github.com/thought-machine/please/tools/please_go/install"
)

var opts = struct {
	Usage string

	PleaseGoInstall struct {
		SrcRoot      string `short:"r" long:"src_root" description:"The src root of the module to inspect" default:"."`
		ModuleName   string `short:"n" long:"module_name" description:"The name of the module" required:"true"`
		ImportConfig string `short:"i" long:"importcfg" description:"The import config for the modules dependencies" required:"true"`
		LDFlags      string `short:"l" long:"ld_flags" description:"The file to write linker flags to" default:"LD_FLAGS"`
		GoTool       string `short:"g" long:"go_tool" description:"The location of the go binary" default:"go"`
		CCTool       string `short:"c" long:"cc_tool" description:"The c compiler to use"`
		Out          string `short:"o" long:"out" description:"The output directory to put compiled artifacts in" required:"true"`
		Args         struct {
			Packages []string `positional-arg-name:"packages" description:"The packages to compile"`
		} `positional-args:"true" required:"true"`
	} `command:"install" alias:"i" description:"Compile a go module similarly to 'go install'"`
	Test struct {
		Dir        string        `short:"d" long:"dir" description:"Directory to search for Go package files for coverage"`
		Exclude    []string      `short:"x" long:"exclude" default:"third_party/go" description:"Directories to exclude from search"`
		Output     string        `short:"o" long:"output" description:"Output filename" required:"true"`
		Package    string        `short:"p" long:"package" description:"Package containing this test" env:"PKG_DIR"`
		ImportPath string        `short:"i" long:"import_path" description:"Full import path to the package"`
		Benchmark  bool          `short:"b" long:"benchmark" description:"Whether to run benchmarks instead of tests"`
		Args       struct {
			Sources []string `positional-arg-name:"sources" description:"Test source files" required:"true"`
		} `positional-args:"true" required:"true"`
	} `command:"testmain" alias:"t" description:"Generates a go main package to run the tests in a package."`
}{
	Usage: `
please-go is used by the go build rules to compile and test go modules and packages. 

Unlike 'go build', this tool doesn't rely on the go path or modules to find packages. Instead it takes in
a go import config just like 'go tool compile/link -importcfg'.
`,
}

var subCommands = map[string]func() int{
	"install": func() int {
		pleaseGoInstall := install.New(
			opts.PleaseGoInstall.SrcRoot,
			opts.PleaseGoInstall.ModuleName,
			opts.PleaseGoInstall.ImportConfig,
			opts.PleaseGoInstall.LDFlags,
			opts.PleaseGoInstall.GoTool,
			opts.PleaseGoInstall.CCTool,
			opts.PleaseGoInstall.Out,
		)
		if err := pleaseGoInstall.Install(opts.PleaseGoInstall.Args.Packages); err != nil {
			log.Fatal(err)
		}
		return 0
	},
	"testmain": func() int {
		test.PleaseGoTest(
			opts.Test.Dir,
			opts.Test.ImportPath,
			opts.Test.Package,
			opts.Test.Output,
			opts.Test.Args.Sources,
			opts.Test.Exclude,
			opts.Test.Benchmark,
		)
		return 0
	},
}

func main() {
	command := flags.ParseFlagsOrDie("please-go", &opts)
	os.Exit(subCommands[command]())
}
