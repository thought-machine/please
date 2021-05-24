package query

import (
	"fmt"

	"github.com/thought-machine/please/src/core"
)

// WhatInputs prints the targets with the provided files as sources
// The targets are printed in the same order as the provided files, separated by a newline
// Use printFiles to additionally echo the files themselves (i.e. print <file> <target>)
func WhatInputs(graph *core.BuildGraph, files []string, local, hidden, printFiles bool) {
	targets := graph.AllTargets()

	for _, file := range files {
		if inputLabels := whatInputs(graph, targets, file, local, hidden); len(inputLabels) > 0 {
			for _, label := range inputLabels {
				if printFiles {
					fmt.Printf("%s ", file)
				}
				fmt.Printf("%s\n", label)
			}
		} else {
			if printFiles {
				fmt.Printf("%s Error: Not a source to any current target\n", file)
			} else {
				fmt.Printf("Error: '%s' is not a source to any current target\n", file)
			}
		}
	}
}

func whatInputs(graph *core.BuildGraph, targets []*core.BuildTarget, file string, local, hidden bool) []core.BuildLabel {
	inputTargets := make(map[*core.BuildTarget]struct{})

	for _, target := range targets {
		var sources []string
		if local {
			sources = target.AllLocalSources()
		} else {
			sources = target.AllSourcePaths(graph)
		}

		for _, source := range sources {
			if source == file {
				inputTargets[resolveTarget(graph, target, !hidden)] = struct{}{}
			}
		}
	}

	ret := make([]core.BuildLabel, 0, len(inputTargets))
	for target := range inputTargets {
		ret = append(ret, target.Label)
	}

	return ret
}

// Resolves targets based on whether we are looking for the parent one or not
func resolveTarget(graph *core.BuildGraph, target *core.BuildTarget, parent bool) *core.BuildTarget {
	if !parent {
		return target
	}

	parentTarget := target.Parent(graph)
	if parentTarget != nil {
		return parentTarget
	}

	return target
}
