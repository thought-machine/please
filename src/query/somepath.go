package query

import "fmt"

import "core"

// SomePath finds and returns a path between two targets.
// Useful for a "why on earth do I depend on this thing" type query.
func SomePath(graph *core.BuildGraph, label1 core.BuildLabel, label2 core.BuildLabel) {
	// Awkwardly either target can be :all. This is an extremely useful idiom though so despite
	// trickiness is worth supporting.
	// Of course this calculation is also quadratic but it's not very obvious how to avoid that.
	if label1.IsAllTargets() {
		for _, target := range graph.PackageOrDie(label1.PackageName).Targets {
			if querySomePath1(graph, target, label2, false) {
				return
			}
		}
		fmt.Printf("Couldn't find any dependency path between %s and %s\n", label1, label2)
	} else {
		querySomePath1(graph, graph.TargetOrDie(label1), label2, true)
	}
}

func querySomePath1(graph *core.BuildGraph, target1 *core.BuildTarget, label2 core.BuildLabel, print bool) bool {
	// Now we do the same for label2.
	if label2.IsAllTargets() {
		for _, target2 := range graph.PackageOrDie(label2.PackageName).Targets {
			if querySomePath2(graph, target1, target2, false) {
				return true
			}
		}
		return false
	}
	return querySomePath2(graph, target1, graph.TargetOrDie(label2), print)
}

func querySomePath2(graph *core.BuildGraph, target1, target2 *core.BuildTarget, print bool) bool {
	if !printSomePath(graph, target1, target2) && !printSomePath(graph, target2, target1) {
		if print {
			fmt.Printf("Couldn't find any dependency path between %s and %s\n", target1.Label, target2.Label)
		}
		return false
	}
	return true
}

// This is just a simple DFS through the graph.
func printSomePath(graph *core.BuildGraph, target1, target2 *core.BuildTarget) bool {
	if target1 == target2 {
		fmt.Printf("Found path:\n  %s\n", target1.Label)
		return true
	}
	for _, target := range graph.ReverseDependencies(target2) {
		if printSomePath(graph, target1, target) {
			if target2.Parent(graph) != target {
				fmt.Printf("  %s\n", target2.Label)
			}
			return true
		}
	}
	return false
}
