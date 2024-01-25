package query

import (
	"fmt"
	"sort"

	"github.com/thought-machine/please/src/core"
)

// WhatInputs prints the targets with the provided files as sources
// The targets are printed in the same order as the provided files, separated by a newline
// Use printFiles to additionally echo the files themselves (i.e. print <file> <target>)
func WhatInputs(graph *core.BuildGraph, files []string, hidden, printFiles, ignoreUnknown bool) {
	targets := graph.AllTargets()

	for _, file := range files {
		if inputLabels := whatInputs(targets, file, hidden); len(inputLabels) > 0 {
			for _, label := range inputLabels {
				if printFiles {
					fmt.Printf("%s ", file)
				}
				fmt.Printf("%s\n", label)
			}
		} else if !ignoreUnknown {
			log.Fatalf("%s is not a source to any current target", file)
		}
	}
}

func whatInputs(targets []*core.BuildTarget, file string, hidden bool) []core.BuildLabel {
	labels := make(map[core.BuildLabel]struct{})
	for _, target := range targets {
		for _, source := range target.AllLocalSourcePaths() {
			if source == file {
				label := target.Label
				if !hidden {
					label = target.Label.Parent()
				}
				labels[label] = struct{}{}
			}
		}
	}

	ret := make(core.BuildLabels, 0, len(labels))
	for label := range labels {
		ret = append(ret, label)
	}
	sort.Sort(ret)

	return ret
}
