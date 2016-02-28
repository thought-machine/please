// Implementation of the BuildInput interface for simple cases of files in the local package.

package core

import "fmt"
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

func (label FileLabel) Label() *BuildLabel {
	return nil
}

func (label FileLabel) String() string {
	return label.File
}

// Similar to above but used for collecting a single output of another file.
type BuildFileLabel struct {
	// Target the label comes from
	BuildLabel BuildLabel
	// Name of the file
	File string
}

func (label BuildFileLabel) Paths(graph *BuildGraph) []string {
	return []string{path.Join(label.BuildLabel.PackageName, label.File)}
}

func (label BuildFileLabel) FullPaths(graph *BuildGraph) []string {
	return []string{path.Join(graph.TargetOrDie(label.BuildLabel).OutDir(), label.File)}
}

func (label BuildFileLabel) Label() *BuildLabel {
	return &label.BuildLabel
}

func (label BuildFileLabel) String() string {
	return fmt.Sprintf("%s:%s", label.BuildLabel, label.File)
}

// SystemFileLabel represents an absolute system dependency, which is not managed by the build system.
type SystemFileLabel struct {
	Path string
}

func (label SystemFileLabel) Paths(graph *BuildGraph) []string {
	return []string{label.Path}
}

func (label SystemFileLabel) FullPaths(graph *BuildGraph) []string {
	return []string{label.Path}
}

func (label SystemFileLabel) Label() *BuildLabel {
	return nil
}

func (label SystemFileLabel) String() string {
	return label.Path
}

