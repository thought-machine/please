package query

import (
	"core"
	"fmt"
	"cli"
)

// Filter takes the list of BuildLabels and checks which ones match the label selectors passed in.
// If no includeLabels are provided, the default behaviour is to include all labels.
func Filter(graph *core.BuildGraph, labels core.BuildLabels, includeLabels []string, excludeLabels []string) {
	matcher := func(target *core.BuildTarget) bool {
		include := len(includeLabels) == 0
		for _, l  := range target.Labels {
			if cli.ContainsString(l, includeLabels) {
				include = true
			}
		}
		for _, l  := range target.Labels {
			if cli.ContainsString(l, excludeLabels) {
				include = false
			}
		}
		return include
	}

	for _, label := range labels {
		if matcher(graph.TargetOrDie(label)) {
			fmt.Println(label)
		}
	}
}