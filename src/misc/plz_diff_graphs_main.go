// plz_diff_graphs is a small utility to take the JSON representation of two build graphs
// (as output from 'plz query graph') and produce a list of targets that have changed
// between the two.
//
// Note that the 'ordering' of the two graphs matters, hence their labels 'before' and 'after';
// the operation is non-commutative because targets that are added appear and those deleted do not.
//
// It also accepts a list of filenames that have changed and invalidates targets appropriately.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"cli"
	"misc"
)

var opts struct {
	Verbosity    int      `short:"v" long:"verbosity" description:"Verbosity of output (higher number = more output, default 2 -> notice, warnings and errors only)" default:"2"`
	Before       string   `short:"b" long:"before" required:"true" description:"File containing build graph before changes."`
	After        string   `short:"a" long:"after" required:"true" description:"File containing build graph after changes."`
	Include      []string `short:"i" long:"include" description:"Label of targets to include."`
	Exclude      []string `short:"e" long:"exclude" description:"Label of targets to exclude." default:"manual"`
	NoRecurse    bool     `long:"norecurse" description:"Don't recurse into dependencies of rules to see if they've changed"`
	ChangedFiles struct {
		Files []string `positional-arg-name:"files" description:"Files that have changed. - to read from stdin."`
	} `positional-args:"true"`
}

func readStdin() []string {
	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	trimmed := strings.TrimSpace(string(stdin))
	if trimmed == "" {
		return []string{}
	}
	ret := strings.Split(trimmed, "\n")
	for i, s := range ret {
		ret[i] = strings.TrimSpace(s)
	}
	return ret
}

func main() {
	cli.ParseFlagsOrDie("Please graph differ", "5.5.0", &opts)
	cli.InitLogging(opts.Verbosity)
	before := misc.ParseGraphOrDie(opts.Before)
	after := misc.ParseGraphOrDie(opts.After)
	if len(opts.ChangedFiles.Files) == 1 && opts.ChangedFiles.Files[0] == "-" {
		opts.ChangedFiles.Files = readStdin()
	}
	for _, label := range misc.DiffGraphs(before, after, opts.ChangedFiles.Files, opts.Include, opts.Exclude, !opts.NoRecurse) {
		fmt.Printf("%s\n", label)
	}
}
