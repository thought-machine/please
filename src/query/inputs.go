package query

import (
	"fmt"
	"maps"
	"sort"

	"github.com/thought-machine/please/src/core"
)

// TargetInputs prints all inputs for a single target.
func TargetInputs(graph *core.BuildGraph, labels []core.BuildLabel) {
	inputPaths := map[string]bool{}
	for _, label := range labels {
		for sourcePath := range core.IterInputPaths(graph, graph.TargetOrDie(label)) {
			inputPaths[sourcePath] = true
		}
	}

	keys := maps.Keys(inputPaths)
	sort.Strings(keys)
	for _, path := range keys {
		fmt.Printf("%s\n", path)
	}
}
