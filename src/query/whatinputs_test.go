package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func addNewTarget(graph *core.BuildGraph, pkg *core.Package, targetName string, sources []core.BuildInput, outs []string, deps []core.BuildLabel) *core.BuildTarget {
	target := core.NewBuildTarget(core.NewBuildLabel(pkg.Name, targetName))
	for _, source := range sources {
		target.AddSource(source)
	}
	for _, out := range outs {
		target.AddOutput(out)
		pkg.MustRegisterOutput(out, target)
	}
	for _, dep := range deps {
		target.AddDependency(dep)
	}
	if err := target.ResolveDependencies(graph); err != nil {
		log.Fatalf("Unable to resolve dependencies for %s - %s", target, err)
	}
	pkg.AddTarget(target)
	graph.AddTarget(target)

	return target
}

func TestWhatInputsSimple(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	addNewTarget(graph, pkg1, "target1", []core.BuildInput{fileSource}, nil, nil)
	graph.AddPackage(pkg1)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", false, false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsIncompleteSourcePath(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	addNewTarget(graph, pkg1, "target1", []core.BuildInput{fileSource}, nil, nil)
	graph.AddPackage(pkg1)

	inputLabels := whatInputs(graph, graph.AllTargets(), "file1.txt", false, false)
	assert.Equal(t, []core.BuildLabel{}, inputLabels)
}

func TestWhatInputsMultipleTargetsSamePackage(t *testing.T) {
	graph := core.NewGraph()
	pkg := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg.Name}
	addNewTarget(graph, pkg, "target1", []core.BuildInput{fileSource}, nil, nil)
	addNewTarget(graph, pkg, "target2", []core.BuildInput{fileSource}, nil, nil)
	graph.AddPackage(pkg)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", false, false)
	assert.Len(t, inputLabels, 2)
	assert.Contains(t, inputLabels, core.BuildLabel{PackageName: "package1", Name: "target1"})
	assert.Contains(t, inputLabels, core.BuildLabel{PackageName: "package1", Name: "target2"})
}

func TestWhatInputsMultipleTargetsDifferentPackages(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	target1 := addNewTarget(graph, pkg1, "target1", []core.BuildInput{fileSource}, []string{"file1.txt"}, nil)
	graph.AddPackage(pkg1)
	pkg2 := core.NewPackage("package2")
	addNewTarget(graph, pkg2, "target2", []core.BuildInput{target1.Label}, nil, nil)
	graph.AddPackage(pkg2)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", false, false)
	assert.Len(t, inputLabels, 2)
	assert.Contains(t, inputLabels, core.BuildLabel{PackageName: "package1", Name: "target1"})
	assert.Contains(t, inputLabels, core.BuildLabel{PackageName: "package2", Name: "target2"})
}

func TestWhatInputsLocalTargetsOnly(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	target1 := addNewTarget(graph, pkg1, "target1", []core.BuildInput{fileSource}, []string{"file1.txt"}, nil)
	graph.AddPackage(pkg1)
	pkg2 := core.NewPackage("package2")
	addNewTarget(graph, pkg2, "target2", []core.BuildInput{target1.Label}, nil, nil)
	graph.AddPackage(pkg2)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", true, false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsInternalTargetsExist1(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	// Same input to both targets
	addNewTarget(graph, pkg1, "target1", []core.BuildInput{fileSource}, nil, nil)
	addNewTarget(graph, pkg1, "_target1#one", []core.BuildInput{fileSource}, nil, nil)
	graph.AddPackage(pkg1)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", false, false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsInternalTargetsExist2(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	// Only internal target contains the source
	addNewTarget(graph, pkg1, "target1", nil, nil, nil)
	addNewTarget(graph, pkg1, "_target1#one", []core.BuildInput{fileSource}, nil, nil)
	graph.AddPackage(pkg1)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", false, false)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, inputLabels)
}

func TestWhatInputsInternalTargetsSelected1(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	// Same input to both targets
	addNewTarget(graph, pkg1, "target1", []core.BuildInput{fileSource}, nil, nil)
	addNewTarget(graph, pkg1, "_target1#one", []core.BuildInput{fileSource}, nil, nil)
	graph.AddPackage(pkg1)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", false, true)
	assert.Len(t, inputLabels, 2)
	assert.Contains(t, inputLabels, core.BuildLabel{PackageName: "package1", Name: "target1"})
	assert.Contains(t, inputLabels, core.BuildLabel{PackageName: "package1", Name: "_target1#one"})
}

func TestWhatInputsInternalTargetsSelected2(t *testing.T) {
	graph := core.NewGraph()
	pkg1 := core.NewPackage("package1")
	fileSource := core.FileLabel{File: "file1.txt", Package: pkg1.Name}
	// Only internal target contains the source
	addNewTarget(graph, pkg1, "target1", nil, nil, nil)
	addNewTarget(graph, pkg1, "_target1#one", []core.BuildInput{fileSource}, nil, nil)
	graph.AddPackage(pkg1)

	inputLabels := whatInputs(graph, graph.AllTargets(), "package1/file1.txt", false, true)
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "_target1#one"}}, inputLabels)
}
