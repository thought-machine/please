package query

import "core"
import "fmt"

// ReverseDeps For each input label, finds all targets which depend upon it.
func ReverseDeps(graph *core.BuildGraph, labels []core.BuildLabel) {

	uniqueTargets := make(map[core.BuildLabel]struct{})

	for _, label := range labels {
		for _, target := range graph.ReverseDependencies(graph.TargetOrDie(label)) {
			uniqueTargets[target.Label] = struct {}{}
		}
	}

	for target := range uniqueTargets {
		fmt.Printf("%s\n", target)
	}
}
