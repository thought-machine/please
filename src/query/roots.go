package query

import (
	"github.com/thought-machine/please/src/core"
	"fmt"
	"sort"
)

// Roots returns build labels with no dependents from the given list.
// i.e. if `labels` contains `A` and `B` such that `A` depends-on `B` (possibly through some indirect path)
// only `B` will be output.
// This does not perform an ordering of the labels, but theoretically you could call this method repeatedly with a
// shrinking set of labels.
func Roots(graph *core.BuildGraph, labels core.BuildLabels) {
	// allSeenTargets is every single reverse dependency of the passed in labels.
	allSeenTargets := map[*core.BuildTarget]struct{}{}
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		// Minor efficiency, if we've seen this target already there's no point iterating it
		// As any of its reverse dependencies will also have already been found.
		_, ok := allSeenTargets[target]
		if ok {
			continue
		}

		targets := map[*core.BuildTarget]struct{}{}
		uniqueReverseDependencies(graph, target, targets)
		// We need to remove the current target from the returned list as it is a false positive at this point.
		delete(targets, target)
		for parent := range targets {
			allSeenTargets[parent] = struct{}{}
		}
	}
	for parent := range allSeenTargets {
		// See if any of the reverse deps were passed in
		i := indexOf(labels, parent.Label)
		if i != -1 {
			// If so, we know it must not be a root, so remove it from the set
			labels[i] = labels[len(labels)-1]
			labels[len(labels)-1] = core.BuildLabel{}
			labels = labels[:len(labels)-1]
		}
	}
	sort.Sort(labels)
	for _, l := range labels {
		fmt.Printf("%s\n", l)
	}
}

func indexOf(labels []core.BuildLabel, label core.BuildLabel) int {
	for i, l := range labels {
		if l == label {
			return i
		}
	}
	return -1
}

func uniqueReverseDependencies(graph *core.BuildGraph, target *core.BuildTarget, targets map[*core.BuildTarget]struct{}) {
	_, ok := targets[target]
	if ok {
		return
	}
	targets[target] = struct{}{}
	// ReverseDependencies are the smaller order collection, so more efficient to iterate.
	for _, child := range graph.ReverseDependencies(target) {
		uniqueReverseDependencies(graph, child, targets)
	}
}
