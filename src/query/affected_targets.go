package query

import "core"
import "fmt"

// AffectedTargets walks over the build graph and identifies all targets that have a transitive
// dependency on the given set of files.
// Targets are filtered by given include / exclude labels and if 'tests' is true only
// test targets will be returned.
func AffectedTargets(graph *core.BuildGraph, files, include, exclude []string, tests, transitive bool) {
	affectedTargets := make(chan *core.BuildTarget, 100)
	done := make(chan bool)

	filePaths := map[string]bool{}
	for _, file := range files {
		filePaths[file] = true
	}

	// Check all the targets to see if any own one of these files
	go func() {
		for _, target := range graph.AllTargets() {
			for _, source := range target.AllSourcePaths(graph) {
				if _, present := filePaths[source]; present {
					affectedTargets <- target
					break
				}
			}
		}
		done <- true
	}()

	// Check all the packages to see if any are defined by these files.
	// This is pretty pessimistic, we have to just assume the whole package is invalidated.
	// A better approach involves using plz query graph and plz_diff_graphs - see that tool
	// for more explanation.
	go func() {
		invalidatePackage := func(pkg *core.Package) {
			for _, target := range pkg.Targets {
				affectedTargets <- target
			}
		}
		for _, pkg := range graph.PackageMap() {
			if _, present := filePaths[pkg.Filename]; present {
				invalidatePackage(pkg)
			} else {
				for _, subinclude := range pkg.Subincludes {
					for _, source := range graph.TargetOrDie(subinclude).AllSourcePaths(graph) {
						if _, present := filePaths[source]; present {
							invalidatePackage(pkg)
							break
						}
					}
				}
			}
		}
		done <- true
	}()

	go handleAffectedTargets(graph, affectedTargets, done, include, exclude, tests, transitive)

	<-done
	<-done
	close(affectedTargets)
	<-done
}

func handleAffectedTargets(graph *core.BuildGraph, affectedTargets <-chan *core.BuildTarget, done chan<- bool, include, exclude []string, tests, transitive bool) {
	seenTargets := map[*core.BuildTarget]bool{}

	var inner func(*core.BuildTarget)
	inner = func(target *core.BuildTarget) {
		if !seenTargets[target] {
			seenTargets[target] = true
			if transitive {
				for _, revdep := range graph.ReverseDependencies(target) {
					inner(revdep)
				}
			}
			if (!tests || target.IsTest) && target.ShouldInclude(include, exclude) {
				fmt.Printf("%s\n", target.Label)
			}
		}
	}
	for target := range affectedTargets {
		inner(target)
	}
	done <- true
}
