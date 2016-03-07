package query

import "fmt"

import "core"

func QueryAllTargets(graph *core.BuildGraph, labels []core.BuildLabel, include, exclude []string) {
	for _, label := range labels {
		if target := graph.TargetOrDie(label); target.ShouldInclude(include, exclude) {
			fmt.Printf("%s\n", target.Label)
		}
	}
}
