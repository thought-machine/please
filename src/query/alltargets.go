package query

import (
	"fmt"
	"sort"

	"core"
)

func QueryAllTargets(graph *core.BuildGraph, labels core.BuildLabels, include, exclude []string) {
	sort.Sort(labels)
	for _, label := range labels {
		if target := graph.TargetOrDie(label); target.ShouldInclude(include, exclude) {
			fmt.Printf("%s\n", target.Label)
		}
	}
}
