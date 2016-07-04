package query

import (
	"core"
	"fmt"
	"path"
)

// WhatOuputs prints the target responsible for producing each of the provided files
// The targets are printed in the same order as the provided files, separated by a newline
// Use print_files to additionally echo the files themselves (i.e. print <file> <target>)
func WhatOutputs(graph *core.BuildGraph, files []string, print_files bool) {
	packageMap := filesToLabelMap(graph)
	for _, f := range files {
		if print_files {
			fmt.Printf("%s ", f)
		}
		if build_label, present := packageMap[f]; present {
			fmt.Printf("%s\n", build_label)
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
				artifact_path := path.Join(target.OutDir(), output)
				packageMap[artifact_path] = &target.Label
			}
		}
	}
	return packageMap
}
