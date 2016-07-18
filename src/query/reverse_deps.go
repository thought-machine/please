package query

import (
	"fmt"
	"sort"

	"core"
)

// ReverseDeps For each input label, finds all targets which depend upon it.
func ReverseDeps(graph *core.BuildGraph, labels []core.BuildLabel) {

	uniqueTargets := make(map[core.BuildLabel]struct{})

	for _, label := range labels {
		for _, child := range graph.PackageOrDie(label.PackageName).AllChildren(graph.TargetOrDie(label)) {
			for _, target := range graph.ReverseDependencies(child) {
				if parent := target.Parent(graph); parent != nil {
					uniqueTargets[parent.Label] = struct{}{}
				} else {
					uniqueTargets[target.Label] = struct{}{}
				}
			}
		}
	}

	targets := make(core.BuildLabels, 0, len(uniqueTargets))
	for target := range uniqueTargets {
		targets = append(targets, target)
	}
	sort.Sort(targets)
	for _, target := range targets {
		fmt.Printf("%s\n", target)
	}
}
