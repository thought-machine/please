// Implementation of the BuildInput interface for simple cases of files in the local package.

package core

import "path"
import "strings"

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

func (label FileLabel) nonOutputLabel() *BuildLabel {
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

func (label SystemFileLabel) nonOutputLabel() *BuildLabel {
	return nil
}

func (label SystemFileLabel) String() string {
	return label.Path
}

// NamedOutputLabel represents a reference to a subset of named outputs from a rule.
// The rule must have declared those as a named group.
type NamedOutputLabel struct {
	BuildLabel
	Output string
}

func (label NamedOutputLabel) Paths(graph *BuildGraph) []string {
	return addPathPrefix(graph.TargetOrDie(label.BuildLabel).NamedOutputs(label.Output), label.PackageName)
}

func (label NamedOutputLabel) FullPaths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label.BuildLabel)
	return addPathPrefix(target.NamedOutputs(label.Output), target.OutDir())
}

func (label NamedOutputLabel) LocalPaths(graph *BuildGraph) []string {
	return graph.TargetOrDie(label.BuildLabel).NamedOutputs(label.Output)
}

func (label NamedOutputLabel) Label() *BuildLabel {
	return &label.BuildLabel
}

func (label NamedOutputLabel) nonOutputLabel() *BuildLabel {
	return nil
}

func (label NamedOutputLabel) String() string {
	return label.BuildLabel.String() + "|" + label.Output
}

// TryParseNamedOutputLabel attempts to parse a build output label. It's allowed to just be
// a normal build label as well.
// The syntax is an extension of normal build labels: //package:target|output
func TryParseNamedOutputLabel(target, currentPath string) (BuildInput, error) {
	if index := strings.IndexRune(target, '|'); index != -1 && index != len(target)-1 {
		label, err := TryParseBuildLabel(target[:index], currentPath)
		return NamedOutputLabel{BuildLabel: label, Output: target[index+1:]}, err
	}
	return TryParseBuildLabel(target, currentPath)
}
