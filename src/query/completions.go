package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

type CompletionPackages struct {
	// Pkgs are any subpackages that are valid completions
	Pkgs []string
	// PackageToParse is optionally the package we should include labels from in the completions results
	PackageToParse string
	// NamePrefix is the prefix we should use to match the names of labels in the package.
	NamePrefix string
	// Hidden is whether we should include hidden targets in the results
	Hidden bool
	// IsRoot is whether or not he query matched the root package
	IsRoot bool
}

// CompletePackages produces a set of packages that are valid for a given input
func CompletePackages(config *core.Configuration, query string) *CompletionPackages {
	if !strings.HasPrefix(query, "//") && core.RepoRoot != core.InitialWorkingDir {
		if strings.HasPrefix(query, ":") {
			query = fmt.Sprintf("//%s%s", core.InitialPackagePath, query)
		} else {
			query = "//" + filepath.Join(core.InitialPackagePath, query)
		}
	}
	query = strings.ReplaceAll(query, "\\:", ":")
	isRoot := query == "//" || strings.HasPrefix(query, "//:") || strings.HasPrefix(query, ":")

	if strings.Contains(query, ":") {
		parts := strings.Split(query, ":")
		if len(parts) != 2 {
			log.Fatalf("invalid build label %v", query)
		}
		return &CompletionPackages{
			PackageToParse: strings.TrimLeft(parts[0], "/"),
			NamePrefix:     parts[1],
			Hidden:         strings.HasPrefix(parts[1], "_"),
			IsRoot:         isRoot,
		}
	}

	pkgs, pkg := getPackagesAndPackageToParse(config, query)
	return &CompletionPackages{
		Pkgs:           pkgs,
		PackageToParse: pkg,
		IsRoot:         isRoot,
	}
}

// findPrefixedPackages finds any packages that match a prefix in a directory e.g. src/plz matches src/plz, and
// src/plzinit
func findPrefixedPackages(config *core.Configuration, root, prefix string) []string {
	if root == "" {
		root = "."
	}
	dirs, err := os.ReadDir(root)
	if err != nil {
		log.Fatalf("%v", err)
	}

	var matchedPkgs []string
	for _, d := range dirs {
		if d.IsDir() && strings.HasPrefix(d.Name(), prefix) {
			p := filepath.Join(root, d.Name())
			if containsPackage(config, p) {
				matchedPkgs = append(matchedPkgs, p)
			}
		}
	}

	return matchedPkgs
}

// getPackagesAndPackageToParse returns a list of packages that are possible completions and optionally, the package to
// parse if we should include it's labels as well.
func getPackagesAndPackageToParse(config *core.Configuration, query string) ([]string, string) {
	// Whether we need to include build labels or just the packages in the results
	packageOnly := strings.HasSuffix(query, "/") && query != "//"

	query = strings.Trim(query, "/")
	root := filepath.Join(core.RepoRoot, query)
	currentPackage := query
	prefix := ""
	if info, err := os.Lstat(root); err != nil || !info.IsDir() {
		_, prefix = filepath.Split(root)
		currentPackage = filepath.Dir(query)
	} else if !packageOnly {
		// If we match a package directly but that's also a prefix for another package, we should return those packages
		root, prefix := filepath.Split(query)
		packages := findPrefixedPackages(config, root, prefix)
		if len(packages) > 1 {
			return packages, ""
		}
	}

	pkgs, pkg := getAllCompletions(config, currentPackage, prefix, packageOnly)
	if packageOnly && pkg == currentPackage || !fs.IsPackage(config.Parse.BuildFileName, pkg) {
		return pkgs, ""
	}
	if pkg == "." {
		pkg = ""
	}
	return pkgs, pkg
}

func isExcluded(config *core.Configuration, dir string) bool {
	if dir == "plz-out" {
		return true
	}
	for _, blacklisted := range config.Parse.BlacklistDirs {
		if filepath.Base(dir) == blacklisted {
			return true
		}
	}
	return false
}

// containsPackage does a breadth first search for build files returning when it encounters the first BUILD file
func containsPackage(config *core.Configuration, dir string) bool {
	dirQueue := []string{dir}
	for len(dirQueue) > 0 {
		dir, dirQueue = dirQueue[0], dirQueue[1:]
		if isExcluded(config, dir) {
			continue
		}

		infos, err := os.ReadDir(dir)
		if err != nil {
			log.Fatalf("failed to find subpackages: %v", err)
		}

		for _, info := range infos {
			if info.IsDir() {
				dirQueue = append(dirQueue, filepath.Join(dir, info.Name()))
			}
			if config.IsABuildFile(info.Name()) {
				return true
			}
		}
	}

	return false
}

// getAllCompletions essentially the same as getPackagesAndPackageToParse without the setup
func getAllCompletions(config *core.Configuration, currentPackage, prefix string, skipSelf bool) ([]string, string) {
	packages := findPrefixedPackages(config, currentPackage, prefix)

	// If we match just one package, return all the immediate subpackages, and return the single package we matched
	if len(packages) == 1 {
		if !skipSelf && prefix == "" && fs.IsPackage(config.Parse.BuildFileName, currentPackage) {
			return packages, currentPackage
		}
		pkgs, pkg := getAllCompletions(config, packages[0], "", false)
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
func Completions(graph *core.BuildGraph, completions *CompletionPackages, binary, test, hidden bool) []string {
	labels := labelsInPackage(graph, completions.PackageToParse, completions.NamePrefix, binary, test, hidden)
	// If we're printing binary targets, we might not match any targets in the parsed. If we only matched one other
	// package, we should try and match binary targets in there.
	if binary && len(labels) == 0 && len(completions.Pkgs) == 1 {
		return labelsInPackage(graph, completions.Pkgs[0], completions.NamePrefix, binary, test, hidden)
	}
	return labels
}

func labelsInPackage(graph *core.BuildGraph, packageName, prefix string, binary, test, hidden bool) []string {
	ts := graph.Package(packageName, "").AllTargets()
	ret := make([]string, 0, len(ts))
	for _, target := range ts {
		if !strings.HasPrefix(target.Label.Name, prefix) {
			continue
		}
		if binary && !target.IsBinary {
			continue
		}
		if test && !target.IsTest() {
			continue
		}
		if hidden || !strings.HasPrefix(target.Label.Name, "_") {
			ret = append(ret, target.Label.String())
		}
	}
	if !binary && prefix == "" && len(ret) > 1 {
		ret = append(ret, fmt.Sprintf("//%s:all", packageName))
	}

	return ret
}

// PrintCompletion prints completions relative to the working package, formatting them based on whether the initial
// query was absolute i.e. started with "//"
func PrintCompletion(completion string, abs bool) {
	if abs {
		if strings.HasPrefix(completion, "//") {
			fmt.Println(completion)
		} else {
			fmt.Printf("//%s\n", completion)
		}
	} else {
		fmt.Println(strings.TrimLeft(strings.TrimPrefix(strings.TrimPrefix(completion, "//"), core.InitialPackagePath), "/"))
	}
}
