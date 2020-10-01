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
			if path := somePath(graph, l1, l2); len(path) != 0 {
				fmt.Println("Found path:")
				for _, l := range path {
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

func somePath(graph *core.BuildGraph, label1, label2 core.BuildLabel) []core.BuildLabel {
	if label1 == label2 {
		return []core.BuildLabel{label1}
	}
	for _, dep := range graph.TargetOrDie(label2).DeclaredDependencies() {
		if path := somePath(graph, label1, dep); len(path) != 0 {
			return append([]core.BuildLabel{label1}, path...)
		}
	}
	return nil
}
