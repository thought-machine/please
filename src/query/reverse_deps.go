package query

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
)

// ReverseDeps finds all transitive targets that depend on the set of input labels.
func ReverseDeps(state *core.BuildState, labels []core.BuildLabel, level int, hidden bool) {
	targets := make(map[core.BuildLabel]int, 100)
	getRevDepTransitiveLabels(state, labels, targets, level)
	for target, _ := range targets {
		if hidden || target.Name[0] != '_' {
			fmt.Printf("%s\n", target)
		}
	}
}

func getRevDepTransitiveLabels(state *core.BuildState, labels []core.BuildLabel, done map[core.BuildLabel]int, levelsToGo int) {
	if levelsToGo == 0 {
		return
	}
	for _, l := range getRevDepsLabels(state, labels) {
		// The level starts high and is decremented, so if we saw this target was a lower level, that means we need to
		// reprocess it with more levels to go.
		if doneLevel, present := done[l]; !present || levelsToGo > doneLevel {
			done[l] = levelsToGo
			getRevDepTransitiveLabels(state, []core.BuildLabel{l}, done, levelsToGo-1)
		}
	}
	ret := core.BuildLabels{}
	for label := range done {
		if state.ShouldInclude(state.Graph.TargetOrDie(label)) {
			ret = append(ret, label)
		}
	}
}

// getRevDepsLabels returns a slice of build labels that are the reverse dependencies of the build labels being passed in
func getRevDepsLabels(state *core.BuildState, labels []core.BuildLabel) core.BuildLabels {
	uniqueTargets := map[*core.BuildTarget]struct{}{}

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
		targets = append(targets, target.Label)
	}
	return targets
}
