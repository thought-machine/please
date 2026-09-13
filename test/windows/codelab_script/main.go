// Command codelab_script reduces the codelabs to a plan test/windows/run_codelabs.ps1 can replay.
// See the script package for why it is built the way it is.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/peterebden/go-cli-init/v5/flags"

	"github.com/thought-machine/please/test/windows/codelab_script/script"
)

var opts = struct {
	Sidecar string `long:"sidecar" required:"true" description:"codelab_steps.conf"`
	Format  string `long:"format" default:"plan" choice:"plan" choice:"summary" description:"plan emits the JSON the runner reads; summary prints one line per block, for authoring the sidecar"`
	Out     string `short:"o" long:"out" description:"File to write to; defaults to stdout"`
	Args    struct {
		Codelabs []string `positional-arg-name:"codelabs" required:"true" description:"The codelab .md files"`
	} `positional-args:"true" required:"true"`
}{}

func main() {
	flags.ParseFlagsOrDie("Codelab script", &opts, nil)

	b, err := os.ReadFile(opts.Sidecar)
	if err != nil {
		die("%s", err)
	}
	side, err := script.ParseSidecar(opts.Sidecar, string(b))
	if err != nil {
		die("%s", err)
	}
	var codelabs []script.Codelab
	for _, filename := range opts.Args.Codelabs {
		b, err := os.ReadFile(filename)
		if err != nil {
			die("%s", err)
		}
		codelabs = append(codelabs, script.ParseCodelab(filename, string(b)))
	}

	var out []byte
	if opts.Format == "summary" {
		// Deliberately tolerant: this is how the sidecar gets written, so it has to print
		// the blocks nothing has decided yet instead of stopping at the first one.
		out = []byte(script.Census(codelabs, side))
	} else {
		plan, errs := script.BuildPlan(codelabs, side)
		if len(errs) > 0 {
			for _, err := range errs {
				fmt.Fprintf(os.Stderr, "%s\n\n", err)
			}
			die("%d problem(s) extracting the codelabs; nothing was written", len(errs))
		}
		if out, err = json.MarshalIndent(plan, "", "  "); err != nil {
			die("%s", err)
		}
		out = append(out, '\n')
	}

	if opts.Out == "" {
		os.Stdout.Write(out)
	} else if err := os.WriteFile(opts.Out, out, 0644); err != nil {
		die("%s", err)
	}
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
