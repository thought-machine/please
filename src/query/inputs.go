package query

import "fmt"
import "core"

func QueryTargetInputs(graph *core.BuildGraph, labels []core.BuildLabel) {
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
