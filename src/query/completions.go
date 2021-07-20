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

type CompletionsLabels struct {
	// Pkgs are any subpackages that are valid completions
	Pkgs []string
	// PackageToParse is optionally the package we should include labels from in the completions results
	PackageToParse string
	// NamePrefix is the prefix we should use to match the names of labels in the package.
	NamePrefix string
	// Hidden is whether we should include hidden targets in the results
	Hidden bool
}

// CompletionLabels produces a set of labels that complete a given input.
// The second return value is a set of labels to parse for (since the original set generally won't turn out to exist).
// The last return value is true if one or more of the inputs are a "hidden" target
// (i.e. name begins with an underscore).
func CompletionLabels(config *core.Configuration, query string, repoRoot string) *CompletionsLabels {
	if query == "" {
		pkgs, toParse := getPackagesAndPackageToParse(config, ".", repoRoot)
		return &CompletionsLabels{
			Pkgs: pkgs,
			PackageToParse: toParse,
		}
	}

	query = strings.ReplaceAll(query, "\\:", ":")

	if strings.Contains(query, ":") {
		parts := strings.Split(query, ":")
		if len(parts) != 2 {
			log.Fatalf("invalid build label %v", query)
		}
		return &CompletionsLabels{
			PackageToParse: strings.TrimLeft(parts[0], "/"),
			NamePrefix:     parts[1],
			Hidden:         strings.HasPrefix(parts[1], "_"),
		}
	}

	pkgs, pkg := getPackagesAndPackageToParse(config, query, repoRoot)
	return &CompletionsLabels{
		Pkgs:           pkgs,
		PackageToParse: pkg,
	}
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
		_, prefix = path.Split(root)
		currentPackage = path.Dir(query)
	}

	// TODO(jpoole): We currently walk the entire file tree trying to discover BUILD files whereas we can probably just
	// 	walk until we find the first ones in each branch and build a trie. This seems fast enough for now though.
	allPackages := make([]string, 0, 10)
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

// getAllCompletions essentially the same as getPackagesAndPackageToParse without the setup
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
func Completions(graph *core.BuildGraph, packageName, prefix string, pkgs []string, binary, test, hidden bool) {
	count := printLabelsInPackage(graph, packageName, prefix, binary, test, hidden)
	// If we're printing binary targets, we might not match any targets in the parsed. If we only matched one other
	// package, we should try and match binary targets in there.
	if binary && count == 0 && len(pkgs) == 1 {
		printLabelsInPackage(graph, pkgs[0], prefix, binary, test, hidden)
	} else {
		for _, pkg := range pkgs {
			fmt.Printf("//%s\n", pkg)
		}
	}
}

func printLabelsInPackage(graph *core.BuildGraph, packageName, prefix string, binary, test, hidden bool) int {
	count := 0
	for _, target := range graph.Package(packageName, "").AllTargets() {
		if !strings.HasPrefix(target.Label.Name, prefix) {
			continue
		}
		if binary && !target.IsBinary {
			continue
		}
		if test && !target.IsTest {
			continue
		}
		if hidden || !strings.HasPrefix(target.Label.Name, "_") {
			fmt.Printf("%s\n", target.Label)
			count++
		}
	}
	if !binary && prefix == "" {
		fmt.Printf("//%s:all\n", packageName)
	}

	return count
}
