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
	"fmt"
	"strings"

	"cli"
	"tools/please_maven/maven"
)

var opts = struct {
	Usage        string
	Repositories []string `short:"r" long:"repository" description:"Location of Maven repo" default:"https://repo1.maven.org/maven2"`
	Android      bool     `short:"a" long:"android" description:"Adds https://maven.google.org to repositories for Android deps."`
	Verbosity    int      `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Exclude      []string `short:"e" long:"exclude" description:"Artifacts to exclude from download"`
	Indent       bool     `short:"i" long:"indent" description:"Indent stdout lines appropriately"`
	Optional     []string `short:"o" long:"optional" description:"Optional dependencies to fetch"`
	BuildRules   bool     `short:"b" long:"build_rules" description:"Print individual maven_jar build rules for each artifact"`
	NumThreads   int      `short:"n" long:"num_threads" default:"10" description:"Number of concurrent fetches to perform"`
	LicenceOnly  bool     `short:"l" long:"licence_only" description:"Fetch only the licence of the given package from Maven"`
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
		fmt.Println(strings.Join(maven.AllDependencies(f, opts.Args.Artifacts, opts.NumThreads, opts.Indent, opts.BuildRules), "\n"))
	}
}
