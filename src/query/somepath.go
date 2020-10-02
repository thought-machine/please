package query

import "fmt"

import "github.com/thought-machine/please/src/core"

// SomePath finds and returns a path between two targets, or between one and a set of targets.
// Useful for a "why on earth do I depend on this thing" type query.
func SomePath(graph *core.BuildGraph, from, to []core.BuildLabel) {
	from = expandAllTargets(graph, from)
	to = expandAllTargets(graph, to)
	for _, l1 := range expandAllTargets(graph, from) {
		for _, l2 := range expandAllTargets(graph, to) {
			if path := somePath(graph, graph.TargetOrDie(l1), graph.TargetOrDie(l2)); len(path) != 0 {
				fmt.Println("Found path:")
				for _, l := range filterPath(path) {
					fmt.Printf("  %s\n", l)
				}
				return
			}
		}
	}
	log.Fatalf("Couldn't find any dependency path between those targets")
}

// expandAllTargets expands any :all labels in the given set.
func expandAllTargets(graph *core.BuildGraph, labels []core.BuildLabel) []core.BuildLabel {
	ret := make([]core.BuildLabel, 0, len(labels))
	for _, l := range labels {
		if l.IsAllTargets() {
			for _, t := range graph.PackageOrDie(l).AllTargets() {
				ret = append(ret, t.Label)
			}
		} else {
			ret = append(ret, l)
		}
	}
	return ret
}

func somePath(graph *core.BuildGraph, target1, target2 *core.BuildTarget) []core.BuildLabel {
	// Have to try this both ways around since we don't know which is a dependency of the other.
	if path := somePath2(graph, target1, target2); len(path) != 0 {
		return path
	}
	return somePath2(graph, target2, target1)
}

func somePath2(graph *core.BuildGraph, target1, target2 *core.BuildTarget) []core.BuildLabel {
	if target1.Label == target2.Label {
		return []core.BuildLabel{target1.Label}
	}
	for _, dep := range target1.Dependencies() {
		if path := somePath2(graph, dep, target2); len(path) != 0 {
			return append([]core.BuildLabel{target1.Label}, path...)
		}
	}
	return nil
}

// filterPath filters out any internal targets on a path between two targets.
func filterPath(path []core.BuildLabel) []core.BuildLabel {
	ret := []core.BuildLabel{path[0]}
	last := path[0]
	for _, l := range path {
		if l.Parent() != last {
			ret = append(ret, l)
			last = l
		}
	}
	return ret
}
