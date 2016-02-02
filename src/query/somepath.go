package query

import "fmt"

import "core"

// Finds and returns a path between two targets.
// Useful for a "why on earth do I depend on this thing" type query.
func QuerySomePath(graph *core.BuildGraph, label1 core.BuildLabel, label2 core.BuildLabel) {
	target1 := graph.TargetOrDie(label1)
	target2 := graph.TargetOrDie(label2)
	if !printSomePath(graph, target1, target2) && !printSomePath(graph, target2, target1) {
		fmt.Printf("Couldn't find any dependency path between %s and %s\n", label1, label2)
	}
}

// This is just a simple DFS through the graph.
func printSomePath(graph *core.BuildGraph, target1, target2 *core.BuildTarget) bool {
	if target1 == target2 {
		fmt.Printf("Found path:\n  %s\n", target1.Label)
		return true
	}
	for _, target := range graph.ReverseDependencies(target2) {
		if printSomePath(graph, target1, target) {
			fmt.Printf("  %s\n", target2.Label)
			return true
		}
	}
	return false
}
