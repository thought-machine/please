package query

import (
	"fmt"

	"github.com/thought-machine/please/src/core"
)

// WhatOutputs prints the target responsible for producing each of the provided files
// The targets are printed in the same order as the provided files, separated by a newline
// Use printFiles to additionally echo the files themselves (i.e. print <file> <target>)
func WhatOutputs(graph *core.BuildGraph, files []string, printFiles bool) {
	targets := graph.AllTargets()
	for _, f := range files {
		if t := whatOutputs(targets, f); len(t) > 0 {
			for _, l := range t {
				if printFiles {
					fmt.Printf("%s ", f)
				}
				fmt.Printf("%s\n", l)
			}
		} else {
			if printFiles {
				fmt.Printf("%s Error: Not a product of any current target\n", f)
			} else {
				fmt.Printf("Error: '%s' is not a product of any current target\n", f)
			}
		}
	}
}

func whatOutputs(targets []*core.BuildTarget, file string) []core.BuildLabel {
	ret := []core.BuildLabel{}
	for _, t := range targets {
		for _, output := range t.FullOutputs() {
			if output == file {
				ret = append(ret, t.Label)
			}
		}
	}
	return ret
}
