package query

import "core"
import "fmt"

func QueryDeps(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, label := range labels {
		printTarget(graph, graph.TargetOrDie(label), "")
	}
}

func printTarget(graph *core.BuildGraph, target *core.BuildTarget, indent string) {
	fmt.Printf("%s%s\n", indent, target.Label)
	for _, dep := range target.Dependencies() {
		printTarget(graph, dep, indent+"  ")
	}
}
