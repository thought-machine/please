package query

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/thought-machine/please/src/core"
)

// WhatInputs prints the targets with the provided files as sources
// The targets are printed in the same order as the provided files, separated by a newline
// Use printFiles to additionally echo the files themselves (i.e. print <file> <target>)
func WhatInputs(graph *core.BuildGraph, files []string, hidden, printFiles, ignoreUnknown bool) {
	inputs := whatInputs(graph.AllTargets(), files, hidden)

	for _, file := range files {
		if inputLabels := inputs[file]; len(inputLabels) > 0 {
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

func whatInputs(targets []*core.BuildTarget, files []string, hidden bool) map[string]core.BuildLabels {
	filesMap := make(map[string]map[core.BuildLabel]struct{}, len(files))
	for _, file := range files {
		filesMap[file] = make(map[core.BuildLabel]struct{})
	}
	for _, target := range targets {
		for _, source := range target.AllLocalSourcePaths() {
			if labels, ok := filesMap[source]; ok {
				label := target.Label
				if !hidden {
					label = target.Label.Parent()
				}
				labels[label] = struct{}{}
			}
		}
	}
	ret := make(map[string]core.BuildLabels, len(filesMap))
	for file, labels := range filesMap {
		ret[file] = slices.SortedFunc(maps.Keys(labels), func(a, b core.BuildLabel) int { return cmp.Compare(a.String(), b.String()) })
	}
	return ret
}
