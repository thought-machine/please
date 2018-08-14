package query

import (
			"core"
	"fmt"
	"sort"
)

// ReverseDeps For each input label, finds all targets which depend upon it.
func ReverseDeps(state *core.BuildState, labels []core.BuildLabel) {
	uniqueTargets := make(map[core.BuildLabel]*core.BuildTarget)

	graph := state.Graph
	for _, label := range labels {
		for _, child := range graph.PackageOrDie(label).AllChildren(graph.TargetOrDie(label)) {
			for _, target := range graph.ReverseDependencies(child) {
				if parent := target.Parent(graph); parent != nil {
					uniqueTargets[parent.Label] = parent
				} else {
					uniqueTargets[target.Label] = target
				}
			}
		}
	}
	// Check for anything subincluding this guy too
	for _, pkg := range graph.PackageMap() {
		for _, label := range labels {
			if pkg.HasSubinclude(label) {
				for _, target := range pkg.AllTargets() {
					uniqueTargets[target.Label] = target
				}
			}
		}
	}

	targets := make(core.BuildLabels, 0, len(uniqueTargets))
	for _, label := range labels {
		delete(uniqueTargets, label)
	}
	for _, target := range uniqueTargets {
		if state.ShouldInclude(target) {
			targets = append(targets, target.Label)
		}
	}
	sort.Sort(targets)
	for _, target := range targets {
		fmt.Printf("%s\n", target)
	}
}
