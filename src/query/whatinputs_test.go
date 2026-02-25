package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func addNewTarget(graph *core.BuildGraph, pkg *core.Package, targetName string, sources []core.BuildInput) *core.BuildTarget {
	target := core.NewBuildTarget(core.NewBuildLabel(pkg.Name, targetName))
	for _, source := range sources {
		target.AddSource(source)
	}
	pkg.AddTarget(target)
	graph.AddTarget(target)
	return target
}

func TestWhatInputsSingleTarget(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	addNewTarget(graph, pkg1, "target1", []core.BuildInput{fileSource})
	graph.AddPackage(pkg1)

	inputLabels := whatInputs(graph.AllTargets(), "package1/file1.txt", false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsMultipleTargets(t *testing.T) {
	graph := core.NewGraph()
	pkg := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg.Name}
	addNewTarget(graph, pkg, "target1", []core.BuildInput{fileSource})
	addNewTarget(graph, pkg, "target2", []core.BuildInput{fileSource})
	graph.AddPackage(pkg)

	inputLabels := whatInputs(graph.AllTargets(), "package1/file1.txt", false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}, {PackageName: "package1", Name: "target2"}}, inputLabels)
}

func TestWhatInputsInternalTargetHidden(t *testing.T) {
	graph := core.NewGraph()
	pkg := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg.Name}
	internalTarget := addNewTarget(graph, pkg, "_target1#srcs", []core.BuildInput{fileSource})
	addNewTarget(graph, pkg, "target1", []core.BuildInput{internalTarget.Label})
	graph.AddPackage(pkg)

	inputLabels := whatInputs(graph.AllTargets(), "package1/file1.txt", false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsInternalTargetShown(t *testing.T) {
	graph := core.NewGraph()
	pkg := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg.Name}
	internalTarget := addNewTarget(graph, pkg, "_target1#srcs", []core.BuildInput{fileSource})
	addNewTarget(graph, pkg, "target1", []core.BuildInput{internalTarget.Label})
	graph.AddPackage(pkg)

	inputLabels := whatInputs(graph.AllTargets(), "package1/file1.txt", true)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "_target1#srcs"}}, inputLabels)
}

func TestWhatInputsSourceBothTargets(t *testing.T) {
	graph := core.NewGraph()
	pkg := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg.Name}
	internalTarget := addNewTarget(graph, pkg, "_target1#srcs", []core.BuildInput{fileSource})
	addNewTarget(graph, pkg, "target1", []core.BuildInput{fileSource, internalTarget.Label})
	graph.AddPackage(pkg)

	inputLabels := whatInputs(graph.AllTargets(), "package1/file1.txt", false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsSourceBothTargetsHidden(t *testing.T) {
	graph := core.NewGraph()
	pkg := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg.Name}
	internalTarget := addNewTarget(graph, pkg, "_target1#srcs", []core.BuildInput{fileSource})
	addNewTarget(graph, pkg, "target1", []core.BuildInput{fileSource, internalTarget.Label})
	graph.AddPackage(pkg)

	inputLabels := whatInputs(graph.AllTargets(), "package1/file1.txt", true)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "_target1#srcs"}, {PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsSingleTargetParentUnderscore(t *testing.T) {
	graph := core.NewGraph()
	pkg := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg.Name}
	internalTarget := addNewTarget(graph, pkg, "__target1#srcs", []core.BuildInput{fileSource})
	addNewTarget(graph, pkg, "_target1", []core.BuildInput{fileSource, internalTarget.Label})
	graph.AddPackage(pkg)

	inputLabels := whatInputs(graph.AllTargets(), "package1/file1.txt", false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "_target1"}}, inputLabels)
}

