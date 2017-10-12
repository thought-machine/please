package query

import "fmt"
import "core"

// TargetInputs prints all inputs for a single target.
func TargetInputs(graph *core.BuildGraph, labels []core.BuildLabel) {
	inputPaths := map[string]bool{}
	for _, label := range labels {
		for sourcePath := range core.IterInputPaths(graph, graph.TargetOrDie(label)) {
			inputPaths[sourcePath] = true
		}
	}

	for path := range inputPaths {
		fmt.Printf("%s\n", path)
	}
}
