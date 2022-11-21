// Implementation of the BuildInput interface for simple cases of files in the local package.

package core

import (
	"path"
	"strings"

	"github.com/thought-machine/please/src/fs"
)

// A BuildInput represents some kind of input to a build rule. They can be implemented
// as either a file (in the local package or on the system) or another build rule.
type BuildInput interface {
	// Paths returns a slice of paths to the files of this input.
	Paths(graph *BuildGraph) []string
	// FullPaths is like Paths but includes the leading plz-out/gen directory.
	FullPaths(graph *BuildGraph) []string
	// LocalPaths returns paths within the local package
	LocalPaths(graph *BuildGraph) []string
	// Label returns the build label associated with this input, or false if it doesn't have one.
	Label() (BuildLabel, bool)
	// nonOutputLabel returns the build label associated with this input, or nil if it doesn't have
	// one or is a specific output of a rule.
	// This is fiddly enough that we don't want to expose it outside the package right now.
	nonOutputLabel() (BuildLabel, bool)
	// String returns a string representation of this input
	String() string
}

// FileLabel represents a file in the current package which is directly used by a target.
type FileLabel struct {
	// Name of the file
	File string
	// Name of the package
	Package string
}

// MarshalText implements the encoding.TextMarshaler interface, which makes FileLabel
// usable as map keys in JSON.
func (label FileLabel) MarshalText() ([]byte, error) {
	return []byte(label.String()), nil
}

// Paths returns a slice of paths to the files of this input.
func (label FileLabel) Paths(graph *BuildGraph) []string {
	return []string{path.Join(label.Package, label.File)}
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label FileLabel) FullPaths(graph *BuildGraph) []string {
	return label.Paths(graph)
}

// LocalPaths returns paths within the local package
func (label FileLabel) LocalPaths(graph *BuildGraph) []string {
	return []string{label.File}
}

// Label returns the build rule associated with this input. For a FileLabel it's always nil.
func (label FileLabel) Label() (BuildLabel, bool) {
	return BuildLabel{}, false
}

func (label FileLabel) nonOutputLabel() (BuildLabel, bool) {
	return BuildLabel{}, false
}

// String returns a string representation of this input.
func (label FileLabel) String() string {
	return label.File
}

// A SubrepoFileLabel represents a file in the current package within a subrepo.
type SubrepoFileLabel struct {
	// Name of the file
	File string
	// Name of the package
	Package string
	// The full path, including the subrepo root.
	FullPackage string
}

// MarshalText implements the encoding.TextMarshaler interface, which makes SubrepoFileLabel
// usable as map keys in JSON.
func (label SubrepoFileLabel) MarshalText() ([]byte, error) {
	return []byte(label.String()), nil
}

// Paths returns a slice of paths to the files of this input.
func (label SubrepoFileLabel) Paths(graph *BuildGraph) []string {
	return []string{path.Join(label.Package, label.File)}
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label SubrepoFileLabel) FullPaths(graph *BuildGraph) []string {
	return []string{path.Join(label.FullPackage, label.File)}
}

// LocalPaths returns paths within the local package
func (label SubrepoFileLabel) LocalPaths(graph *BuildGraph) []string {
	return []string{label.File}
}

// Label returns the build rule associated with this input. For a SubrepoFileLabel it's always nil.
func (label SubrepoFileLabel) Label() (BuildLabel, bool) {
	return BuildLabel{}, false
}

func (label SubrepoFileLabel) nonOutputLabel() (BuildLabel, bool) {
	return BuildLabel{}, false
}

// String returns a string representation of this input.
func (label SubrepoFileLabel) String() string {
	return label.File
}

// NewFileLabel returns either a FileLabel or SubrepoFileLabel as appropriate.
func NewFileLabel(src string, pkg *Package) BuildInput {
	src = strings.TrimRight(src, "/")
	if pkg.Subrepo != nil {
		return SubrepoFileLabel{
			File:        src,
			Package:     pkg.Name,
			FullPackage: pkg.Subrepo.Dir(pkg.Name),
		}
	}
	return FileLabel{File: src, Package: pkg.Name}
}

// SystemFileLabel represents an absolute system dependency, which is not managed by the build system.
type SystemFileLabel struct {
	Path string
}

// MarshalText implements the encoding.TextMarshaler interface, which makes SystemFileLabel
// usable as map keys in JSON.
func (label SystemFileLabel) MarshalText() ([]byte, error) {
	return []byte(label.String()), nil
}

// Paths returns a slice of paths to the files of this input.
func (label SystemFileLabel) Paths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label SystemFileLabel) FullPaths(graph *BuildGraph) []string {
	return []string{fs.ExpandHomePath(label.Path)}
}

// LocalPaths returns paths within the local package
func (label SystemFileLabel) LocalPaths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

// Label returns the build rule associated with this input. For a SystemFileLabel it's always nil.
func (label SystemFileLabel) Label() (BuildLabel, bool) {
	return BuildLabel{}, false
}

func (label SystemFileLabel) nonOutputLabel() (BuildLabel, bool) {
	return BuildLabel{}, false
}

// String returns a string representation of this input.
func (label SystemFileLabel) String() string {
	return label.Path
}

// SystemPathLabel represents system dependency somewhere on PATH, which is not managed by the build system.
type SystemPathLabel struct {
	Name string
	Path []string
}

// MarshalText implements the encoding.TextMarshaler interface, which makes SystemPathLabel
// usable as map keys in JSON.
func (label SystemPathLabel) MarshalText() ([]byte, error) {
	return []byte(label.String()), nil
}

// Paths returns a slice of paths to the files of this input.
func (label SystemPathLabel) Paths(graph *BuildGraph) []string {
	return label.FullPaths(graph)
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label SystemPathLabel) FullPaths(graph *BuildGraph) []string {
	// non-specified paths like "bash" are turned into absolute ones based on plz's PATH.
	// awkwardly this means we can't use the builtin exec.LookPath because the current
	// environment variable isn't necessarily the same as what's in our config.
	tool, err := LookPath(label.Name, label.Path)
	if err != nil {
		// This is a bit awkward, we can't signal an error here sensibly.
		panic(err)
	}
	return []string{tool}
}

// LocalPaths returns paths within the local package
func (label SystemPathLabel) LocalPaths(graph *BuildGraph) []string {
	return []string{label.Name}
}

// Label returns the build rule associated with this input. For a SystemPathLabel it's always nil.
func (label SystemPathLabel) Label() (BuildLabel, bool) {
	return BuildLabel{}, false
}

func (label SystemPathLabel) nonOutputLabel() (BuildLabel, bool) {
	return BuildLabel{}, false
}

// String returns a string representation of this input.
func (label SystemPathLabel) String() string {
	return label.Name
}

// AnnotatedOutputLabel represents a build label with an annotation e.g. //foo:bar|baz where baz constitutes the
// annotation. This can be used to target a named output of this rule when depended on or an entry point when used in
// the context of tools.
type AnnotatedOutputLabel struct {
	BuildLabel
	Annotation string
}

// MarshalText implements the encoding.TextMarshaler interface, which makes AnnotatedOutputLabel
// usable as map keys in JSON.
func (label AnnotatedOutputLabel) MarshalText() ([]byte, error) {
	return []byte(label.String()), nil
}

// Paths returns a slice of paths to the files of this input.
func (label AnnotatedOutputLabel) Paths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label.BuildLabel)
	if _, ok := target.EntryPoints[label.Annotation]; ok {
		return label.BuildLabel.Paths(graph)
	}
	return addPathPrefix(target.NamedOutputs(label.Annotation), label.PackageName)
}

// FullPaths is like Paths but includes the leading plz-out/gen directory.
func (label AnnotatedOutputLabel) FullPaths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label.BuildLabel)
	if _, ok := target.EntryPoints[label.Annotation]; ok {
		return label.BuildLabel.FullPaths(graph)
	}
	return addPathPrefix(target.NamedOutputs(label.Annotation), target.OutDir())
}

// LocalPaths returns paths within the local package
func (label AnnotatedOutputLabel) LocalPaths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label.BuildLabel)
	if _, ok := target.EntryPoints[label.Annotation]; ok {
		return label.BuildLabel.LocalPaths(graph)
	}
	return target.NamedOutputs(label.Annotation)
}

// Label returns the build rule associated with this input. For a AnnotatedOutputLabel it's always non-nil.
func (label AnnotatedOutputLabel) Label() (BuildLabel, bool) {
	return label.BuildLabel, true
}

func (label AnnotatedOutputLabel) nonOutputLabel() (BuildLabel, bool) {
	return BuildLabel{}, false
}

// String returns a string representation of this input.
func (label AnnotatedOutputLabel) String() string {
	if label.Annotation == "" {
		return label.BuildLabel.String()
	}
	return label.BuildLabel.String() + "|" + label.Annotation
}

// UnmarshalFlag unmarshals a build label from a command line flag. Implementation of flags.Unmarshaler interface.
func (label *AnnotatedOutputLabel) UnmarshalFlag(value string) error {
	annotation := ""
	if strings.Count(value, "|") == 1 {
		parts := strings.Split(value, "|")
		value = parts[0]
		annotation = parts[1]
	}

	l := &BuildLabel{}
	if err := l.UnmarshalFlag(value); err != nil {
		return err
	}
	label.BuildLabel = *l
	label.Annotation = annotation
	return nil
}

// A URLLabel represents a remote input that's defined by a URL.
type URLLabel string

// Paths returns an empty slice always (since there are no real source paths)
func (label URLLabel) Paths(graph *BuildGraph) []string {
	return nil
}

// FullPaths returns an empty slice always (since there are no real source paths)
func (label URLLabel) FullPaths(graph *BuildGraph) []string {
	return nil
}

// LocalPaths returns an empty slice always (since there are no real source paths)
func (label URLLabel) LocalPaths(graph *BuildGraph) []string {
	return nil
}

// Label returns the build rule associated with this input. For a URLLabel it's always nil.
func (label URLLabel) Label() (BuildLabel, bool) {
	return BuildLabel{}, false
}

func (label URLLabel) nonOutputLabel() (BuildLabel, bool) {
	return BuildLabel{}, false
}

// String returns a string representation of this input.
func (label URLLabel) String() string {
	return string(label)
}
