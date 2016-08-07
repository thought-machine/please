package query

import (
	"fmt"
	"os"
	"path"
	"strings"

	"core"
	"utils"
)

// QueryCompletionLabels produces a set of labels that complete a given input.
func QueryCompletionLabels(config *core.Configuration, args []string, repoRoot string) []core.BuildLabel {
	if len(args) == 0 {
		queryCompletionPackages(config, ".", repoRoot)
	} else if !strings.Contains(args[0], ":") {
		// Haven't picked a package yet so no parsing is necessary.
		if strings.HasPrefix(args[0], "//") {
			queryCompletionPackages(config, args[0][2:], repoRoot)
		} else {
			queryCompletionPackages(config, args[0], repoRoot)
		}
	}
	if strings.HasSuffix(args[0], ":") {
		args[0] += "all"
	}
	// Bash completion sometimes produces \: instead of just : (see issue #18).
	// We silently fix that here since we've not yet worked out how to fix Bash itself :(
	args[0] = strings.Replace(args[0], "\\:", ":", -1)
	labels := core.ParseBuildLabels([]string{args[0]})
	// Return this label without the trailing bit.
	return []core.BuildLabel{{PackageName: labels[0].PackageName, Name: "all"}}
}

func queryCompletionPackages(config *core.Configuration, query, repoRoot string) {
	root := path.Join(repoRoot, query)
	origRoot := root
	if !core.PathExists(root) {
		root = path.Dir(root)
	}
	packages := []string{}
	for pkg := range utils.FindAllSubpackages(config, root, origRoot) {
		if strings.HasPrefix(pkg, origRoot) {
			packages = append(packages, pkg[len(repoRoot):])
		}
	}
	// If there's only one package, we know it has to be that, but we don't present
	// only one option otherwise bash completion will assume it's that.
	if len(packages) == 1 {
		fmt.Printf("/%s:\n", packages[0])
		fmt.Printf("/%s:all\n", packages[0])
	} else {
		for _, pkg := range packages {
			fmt.Printf("/%s\n", pkg)
		}
	}
	os.Exit(0) // Don't need to run a full-blown parse, get out now.
}

// Queries a set of possible completions for some build labels.
// If 'binary' is true it will complete only targets that are runnable binaries (but not tests).
// If 'test' is true it will similarly complete only targets that are tests.
func QueryCompletions(graph *core.BuildGraph, labels []core.BuildLabel, binary, test bool) {
	for _, label := range labels {
		count := 0
		for _, target := range graph.PackageOrDie(label.PackageName).Targets {
			if (binary && (!target.IsBinary || target.IsTest)) || (test && !target.IsTest) {
				continue
			}
			if !strings.HasPrefix(target.Label.Name, "_") {
				fmt.Printf("%s\n", target.Label)
				count++
			}
		}
		if !binary && count > 1 {
			fmt.Printf("//%s:all\n", label.PackageName)
		}
	}
}
