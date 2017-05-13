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

// BuildOutputLabel represents a single output file from a particular build rule.
type BuildOutputLabel struct {
	BuildLabel
	Output string
}

func (label BuildOutputLabel) Paths(graph *BuildGraph) []string {
	return []string{path.Join(label.PackageName, label.Output)}
}

func (label BuildOutputLabel) FullPaths(graph *BuildGraph) []string {
	return []string{path.Join(graph.TargetOrDie(label.BuildLabel).OutDir(), label.Output)}
}

func (label BuildOutputLabel) LocalPaths(graph *BuildGraph) []string {
	return []string{label.Output}
}

func (label BuildOutputLabel) Label() *BuildLabel {
	return &label.BuildLabel
}

func (label BuildOutputLabel) nonOutputLabel() *BuildLabel {
	return nil
}

func (label BuildOutputLabel) String() string {
	return label.BuildLabel.String() + "|" + label.Output
}

// TryParseBuildOutputLabel attempts to parse a build output label. It's allowed to just be
// a normal build label as well.
func TryParseBuildOutputLabel(target, currentPath string) (BuildInput, error) {
	if index := strings.IndexRune(target, '|'); index != -1 && index != len(target)-1 {
		label, err := TryParseBuildLabel(target[:index], currentPath)
		return BuildOutputLabel{BuildLabel: label, Output: target[index+1:]}, err
	}
	return TryParseBuildLabel(target, currentPath)
}
