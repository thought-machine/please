package query

import (
	"github.com/thought-machine/please/src/core"
	"fmt"
	"path"
)

// WhatOutputs prints the target responsible for producing each of the provided files
// The targets are printed in the same order as the provided files, separated by a newline
// Use printFiles to additionally echo the files themselves (i.e. print <file> <target>)
func WhatOutputs(graph *core.BuildGraph, files []string, printFiles bool) {
	packageMap := filesToLabelMap(graph)
	for _, f := range files {
		if printFiles {
			fmt.Printf("%s ", f)
		}
		if buildLabel, present := packageMap[f]; present {
			fmt.Printf("%s\n", buildLabel)
		} else {
			// # TODO(dimitar): is this a good way to handle unknown files?
			fmt.Println("Error: the file is not a product of any current target")
		}
	}
}

func filesToLabelMap(graph *core.BuildGraph) map[string]*core.BuildLabel {
	packageMap := make(map[string]*core.BuildLabel)
	for _, pkg := range graph.PackageMap() {
		for _, target := range pkg.Outputs {
			for _, output := range target.Outputs() {
				artifactPath := path.Join(target.OutDir(), output)
				packageMap[artifactPath] = &target.Label
			}
		}
	}
	return packageMap
}
