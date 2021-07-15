package query

import (
	"fmt"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/utils"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/core"
)

// CompletionLabels produces a set of labels that complete a given input.
// The second return value is a set of labels to parse for (since the original set generally won't turn out to exist).
// The last return value is true if one or more of the inputs are a "hidden" target
// (i.e. name begins with an underscore).
func CompletionLabels(config *core.Configuration, args []string, repoRoot string) ([]core.BuildLabel, []core.BuildLabel, bool) {
	if len(args) == 0 {
		getPackagesAndPackageToParse(config, ".", repoRoot)
		return []core.BuildLabel{{PackageName: "", Name: ""}}, []core.BuildLabel{{PackageName: "", Name: "all"}}, false
	}

	query := strings.ReplaceAll(args[0], "\\:", ":")
	isRootPackage := query == "" || query == "//"

	if strings.Contains(query, ":") {
		parts := strings.Split(query, ":")
		labels := core.ParseBuildLabels([]string{parts[0] + ":all"})
		return []core.BuildLabel{{PackageName: labels[0].PackageName, Name: parts[1]}}, labels, strings.Contains(query, ":_")
	}

	pkgs, pkg := getPackagesAndPackageToParse(config, query, repoRoot)
	for _, p := range pkgs {
		fmt.Printf("//%s\n", p)
	}
	// We matched more than one package so we don't need to complete any actual labels, or we're matching packages only
	// NB: pkg will be "" for the root package so we should match labels then
	if !isRootPackage && pkg == "" {
		return nil, nil, false
	}
	return []core.BuildLabel{{PackageName: pkg, Name: ""}}, []core.BuildLabel{{PackageName: pkg, Name: "all"}}, false
}

// getPackagesAndPackageToParse returns a list of packages that are possible completions and optionally, the package to
// parse if we should include it's labels as well.
func getPackagesAndPackageToParse(config *core.Configuration, query, repoRoot string) ([]string, string) {
	// Whether we need to include build labels or just the packages in the results
	packageOnly := strings.HasSuffix(query, "/") && query != "//"

	query = strings.Trim(query, "/")
	root := path.Join(repoRoot, query)
	currentPackage := query
	prefix := ""
	if !core.PathExists(root) {
		root, prefix = path.Split(root)
		currentPackage = path.Dir(query)
	}

	// TODO(jpoole): We currently walk the entire file tree trying to discover BUILD files whereas we can probably just
	// 	walk until we find the first ones in each branch and build a trie. This seems fast enough for now though.
	var allPackages []string
	for pkg := range utils.FindAllSubpackages(config, currentPackage, "") {
		allPackages = append(allPackages, pkg)
	}
	pkgs, pkg := getAllCompletions(config, currentPackage, prefix, repoRoot, allPackages, packageOnly)
	if packageOnly && pkg == currentPackage || !fs.IsPackage(config.Parse.BuildFileName, pkg) {
		return pkgs, ""
	}
	return pkgs, pkg
}

func containsPackage(dir string, allPackages []string) bool {
	for _, pkg := range allPackages {
		if strings.HasPrefix(pkg, dir) {
			return true
		}
	}
	return false
}

// getAllCompletions essentailly the same as getPackagesAndPackageToParse without the setup
func getAllCompletions(config *core.Configuration, currentPackage, prefix, repoRoot string, allPackages []string, skipSelf bool) ([]string, string) {
	var packages []string
	root := path.Join(repoRoot, currentPackage)

	dirEntries, err := os.ReadDir(root)
	if err != nil {
		log.Fatalf("failed to check for packages: %v", err)
	}

	for _, entry := range dirEntries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			pkgName := filepath.Join(currentPackage, entry.Name())
			if containsPackage(pkgName, allPackages) {
				packages = append(packages, pkgName)
			}
		}
	}


	// If we match just one package, return all the immediate subpackages, and return the single package we matched
	if len(packages) == 1 {
		if !skipSelf && prefix == "" && fs.IsPackage(config.Parse.BuildFileName, currentPackage) {
			return packages, currentPackage
		}
		pkgs, pkg := getAllCompletions(config, packages[0], "", repoRoot, allPackages, false)
		// If we again matched a package exactly, use that one
		if pkg != "" {
			return pkgs, pkg
		}
		return pkgs, packages[0]
	}

	if prefix == "" {
		return packages, currentPackage
	}

	return packages, ""
}

// Completions queries a set of possible completions for some build labels.
// If 'binary' is true it will complete only targets that are runnable binaries (but not tests).
// If 'test' is true it will similarly complete only targets that are tests.
// If 'hidden' is true then hidden targets (i.e. those with names beginning with an underscore)
// will be included as well.
func Completions(graph *core.BuildGraph, labels []core.BuildLabel, binary, test, hidden bool) {
	for _, label := range labels {
		count := 0
		for _, target := range graph.PackageOrDie(label).AllTargets() {
			if !strings.HasPrefix(target.Label.Name, label.Name) {
				continue
			}
			if (binary && (!target.IsBinary || target.IsTest)) || (test && !target.IsTest) {
				continue
			}
			if hidden || !strings.HasPrefix(target.Label.Name, "_") {
				fmt.Printf("%s\n", target.Label)
				count++
			}
		}
		if !binary && ((label.Name != "" && strings.HasPrefix("all", label.Name)) || (label.Name == "" && count > 1)) { //nolint:gocritic
			fmt.Printf("//%s:all\n", label.PackageName)
		}
	}
}
