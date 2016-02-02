package query

import "fmt"

import "core"

func QueryAllTargets(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, target := range graph.AllTargets() {
		fmt.Printf("%s\n", target.Label)
	}
}
