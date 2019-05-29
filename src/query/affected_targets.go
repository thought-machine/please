package query

import "github.com/thought-machine/please/src/core"
import "fmt"

// AffectedTargets walks over the build graph and identifies all targets that have a transitive
// dependency on the given set of files.
func AffectedTargets(state *core.BuildState, files []string, tests, transitive bool) {
	affectedTargets := make(chan *core.BuildTarget, 100)
	done := make(chan bool)

	filePaths := map[string]bool{}
	for _, file := range files {
		filePaths[file] = true
	}

	// Check all the targets to see if any own one of these files
	go func() {
		for _, target := range state.Graph.AllTargets() {
			// TODO(peterebden): this assumption is very crude, revisit.
			if target.Subrepo == nil {
				for _, source := range target.AllSourcePaths(state.Graph) {
					if _, present := filePaths[source]; present {
						affectedTargets <- target
						break
					}
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
			for _, target := range pkg.AllTargets() {
				affectedTargets <- target
			}
		}
		for _, pkg := range state.Graph.PackageMap() {
			if _, present := filePaths[pkg.Filename]; present {
				invalidatePackage(pkg)
			} else {
				for _, subinclude := range pkg.Subincludes {
					for _, source := range state.Graph.TargetOrDie(subinclude).AllSourcePaths(state.Graph) {
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

	go handleAffectedTargets(state, affectedTargets, done, tests, transitive)

	<-done
	<-done
	close(affectedTargets)
	<-done
}

func handleAffectedTargets(state *core.BuildState, affectedTargets <-chan *core.BuildTarget, done chan<- bool, tests, transitive bool) {
	seenTargets := map[*core.BuildTarget]bool{}

	var inner func(*core.BuildTarget)
	inner = func(target *core.BuildTarget) {
		if !seenTargets[target] {
			seenTargets[target] = true
			if transitive {
				for _, revdep := range state.Graph.ReverseDependencies(target) {
					inner(revdep)
				}
			}
			if (!tests || target.IsTest) && state.ShouldInclude(target) {
				fmt.Printf("%s\n", target.Label)
			}
		}
	}
	for target := range affectedTargets {
		inner(target)
	}
	done <- true
}
