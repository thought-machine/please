package main

import (
	"log"
	"os"

	"github.com/peterebden/go-cli-init/v4/flags"

	"github.com/thought-machine/please/tools/please_go/install"
)

var opts = struct {
	Usage string

	PleaseGoInstall struct {
		SrcRoot      string `short:"r" long:"src_root" description:"The src root of the module to inspect" default:"."`
		ModuleName   string `short:"n" long:"module_name" description:"The name of the module"`
		ImportConfig string `short:"i" long:"importcfg" description:"The import config for the modules dependencies"`
		LDFlags      string `short:"l" long:"ld_flags" description:"The file to write linker flags to" default:"LD_FLAGS"`
		GoTool       string `short:"g" long:"go_tool" description:"The location of the go binary"`
		CCTool       string `short:"c" long:"cc_tool" description:"The c compiler to use"`
		Out          string `short:"o" long:"out" description:"The output directory to put compiled artifacts in"`
		Args         struct {
			Packages []string `positional-arg-name:"packages" description:"The packages to compile"`
		} `positional-args:"true" required:"true"`
	} `command:"install" alias:"i" description:"Compile a go module similarly to 'go install'"`
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
}

func main() {
	command := flags.ParseFlagsOrDie("please-go", &opts)
	os.Exit(subCommands[command]())
}
