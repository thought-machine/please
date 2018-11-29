package query

import (
	"core"
	"fmt"
	"sort"
)

// ReverseDeps For each input label, finds all targets which depend upon it.
func ReverseDeps(state *core.BuildState, labels []core.BuildLabel) {

	targets := GetRevDepsLabels(state, labels)

	for _, target := range targets {
		fmt.Printf("%s\n", target)
	}
}

// GetRevDepsLabels returns a slice of build labels that are the reverse dependencies of the build labels being passed in
func GetRevDepsLabels(state *core.BuildState, labels []core.BuildLabel) core.BuildLabels {
	uniqueTargets := make(map[*core.BuildTarget]struct{})

	graph := state.Graph
	for _, label := range labels {
		for _, child := range graph.PackageOrDie(label).AllChildren(graph.TargetOrDie(label)) {
			for _, target := range graph.ReverseDependencies(child) {
				if parent := target.Parent(graph); parent != nil {
					uniqueTargets[parent] = struct{}{}
				} else {
					uniqueTargets[target] = struct{}{}
				}
			}
		}
	}
	// Check for anything subincluding this guy too
	for _, pkg := range graph.PackageMap() {
		for _, label := range labels {
			if pkg.HasSubinclude(label) {
				for _, target := range pkg.AllTargets() {
					uniqueTargets[target] = struct{}{}
				}
			}
		}
	}

	targets := make(core.BuildLabels, 0, len(uniqueTargets))
	for _, label := range labels {
		delete(uniqueTargets, graph.TargetOrDie(label))
	}
	for target := range uniqueTargets {
		if state.ShouldInclude(target) {
			targets = append(targets, target.Label)
		}
	}
	sort.Sort(targets)

	return targets
}
