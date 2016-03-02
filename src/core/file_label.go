// Implementation of the BuildInput interface for simple cases of files in the local package.

package core

import "path"

type FileLabel struct {
	// Name of the file
	File string
	// Name of the package
	Package string
}

func (label FileLabel) Paths(graph *BuildGraph) []string {
	return []string{path.Join(label.Package, label.File)}
}

func (label FileLabel) FullPaths(graph *BuildGraph) []string {
	return label.Paths(graph)
}

func (label FileLabel) Label() *BuildLabel {
	return nil
}

func (label FileLabel) String() string {
	return label.File
}
