// Package main implements please_maven, a command-line tool to find dependencies
// from a remote Maven repo (typically Maven Central, but can be others).
//
// This is a fairly non-trivial task since the pom.xml format is complex and
// Maven is basically just a static file server for them. We do our best at
// understanding it.
// Of course other packages exist that can parse it, but we prefer not to use them
// since they're Java, and would require shipping a very large binary, but
// more significantly it did not seem easy to make them behave as we wanted.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/tools/please_maven/maven"
)

var opts = struct {
	Usage        string
	Verbosity    cli.Verbosity `short:"v" long:"verbosity" default:"warning" description:"Verbosity of output (higher number = more output)"`
	Repositories []string      `short:"r" long:"repository" description:"Location of Maven repo" default:"https://repo1.maven.org/maven2"`
	Android      bool          `short:"a" long:"android" description:"Adds https://maven.google.org to repositories for Android deps."`
	Exclude      []string      `short:"e" long:"exclude" description:"Artifacts to exclude from download"`
	Indent       bool          `short:"i" long:"indent" description:"Indent stdout lines appropriately"`
	Optional     []string      `short:"o" long:"optional" description:"Optional dependencies to fetch"`
	BuildRules   bool          `short:"b" long:"build_rules" description:"Print individual maven_jar build rules for each artifact"`
	NumThreads   int           `short:"n" long:"num_threads" default:"10" description:"Number of concurrent fetches to perform"`
	LicenceOnly  bool          `short:"l" long:"licence_only" description:"Fetch only the licence of the given package from Maven"`
	Graph        string        `short:"g" long:"graph" description:"Graph file, as exported from plz query graph. If given then existing dependencies in it will be integrated when using --build_rules."`
	Args         struct {
		Artifacts []maven.Artifact `positional-arg-name:"ids" required:"yes" description:"Maven IDs to fetch (e.g. io.grpc:grpc-all:1.4.0)"`
	} `positional-args:"yes" required:"yes"`
}{
	Usage: `
please_maven is a tool shipped with Please that communicates with Maven repositories
to work out what files to download given a package spec.

Example usage:
please_maven io.grpc:grpc-all:1.1.2
> io.grpc:grpc-auth:1.1.2:src:BSD 3-Clause
> io.grpc:grpc-core:1.1.2:src:BSD 3-Clause
> ...
Its output is similarly in the common Maven artifact format which can be used to create
maven_jar rules in BUILD files. It also outputs some notes on whether sources are
available and what licence the package is under, if it can find it.

Note that it does not do complex cross-package dependency resolution and doesn't
necessarily support every aspect of Maven's pom.xml format, which is pretty hard
to fully grok. The goal is to provide a backend to Please's built-in maven_jars
rule to make adding dependencies easier.
`,
}

func loadGraph(filename string) *maven.Graph {
	if filename == "" {
		return &maven.Graph{}
	} else if filename == "-" {
		return decodeGraph(os.Stdin) // Read from stdin.
	}
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	return decodeGraph(f)
}

func decodeGraph(r io.Reader) *maven.Graph {
	g := &maven.Graph{}
	if err := json.NewDecoder(r).Decode(g); err != nil {
		panic(err)
	}
	return g
}

func main() {
	cli.ParseFlagsOrDie("please_maven", "9.0.3", &opts)
	cli.InitLogging(opts.Verbosity)
	if opts.Android {
		opts.Repositories = append(opts.Repositories, "https://maven.google.com")
	}
	f := maven.NewFetch(opts.Repositories, opts.Exclude, opts.Optional)
	if opts.LicenceOnly {
		for _, artifact := range opts.Args.Artifacts {
			for _, licence := range f.Pom(&artifact).Licences.Licence {
				fmt.Println(licence.Name)
			}
		}
	} else {
		fmt.Println(strings.Join(maven.AllDependencies(f, opts.Args.Artifacts, opts.NumThreads, opts.Indent, opts.BuildRules, loadGraph(opts.Graph)), "\n"))
	}
}
