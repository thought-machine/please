// Implementation of the BuildInput interface for simple cases of files in the local package.

package core

import "path"

// FileLabel represents a file in the current package which is directly used by a target.
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

func (label FileLabel) LocalPaths(graph *BuildGraph) []string {
	return []string{label.File}
}

func (label FileLabel) Label() *BuildLabel {
	return nil
}

func (label FileLabel) String() string {
	return label.File
}

// SystemFileLabel represents an absolute system dependency, which is not managed by the build system.
type SystemFileLabel struct {
	Path string
}

func (label SystemFileLabel) Paths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

func (label SystemFileLabel) FullPaths(graph *BuildGraph) []string {
	return []string{ExpandHomePath(label.Path)}
}

func (label SystemFileLabel) LocalPaths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

func (label SystemFileLabel) Label() *BuildLabel {
	return nil
}

func (label SystemFileLabel) String() string {
	return label.Path
}
