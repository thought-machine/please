package query

import (
	"fmt"
	"math/rand"
	"testing"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
)

func BenchmarkDiffGraphs(b *testing.B) {
	logging.SetLevel(logging.WARNING, "")
	for _, numPackages := range []int{1000, 10000} {
		before := createSyntheticGraph(numPackages)
		summary := Summarise(before)
		after := createSyntheticGraph(numPackages)
		numTargets := len(after.Graph.AllTargets())

		b.Run(fmt.Sprintf("Identical/%d", numTargets), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				diffGraphs(summary, after)
			}
		})

		// Modify roughly 1% of the targets in the 'after' graph.
		changed := createSyntheticGraph(numPackages)
		for i, t := range changed.Graph.AllTargets() {
			if i%100 == 0 {
				t.Command += " && echo changed"
			}
		}
		b.Run(fmt.Sprintf("SomeChanged/%d", numTargets), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				diffGraphs(summary, changed)
			}
		})
	}
}

func BenchmarkSummarise(b *testing.B) {
	logging.SetLevel(logging.WARNING, "")
	for _, numPackages := range []int{1000, 10000, 100000} {
		b.Run(fmt.Sprintf("Size/%d", numPackages), func(b *testing.B) {
			graph := createSyntheticGraph(numPackages)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				Summarise(graph)
			}
		})
	}
}

// createSyntheticGraph creates a build state with a graph that looks vaguely like a real repo.
// It is deterministic, so calling it twice with the same arguments produces equivalent graphs.
// Each package contains a library (with a couple of hidden intermediate rules), a test, and a binary;
// libraries depend on a handful of libraries in earlier packages.
func createSyntheticGraph(numPackages int) *core.BuildState {
	state := core.NewDefaultBuildState()
	r := rand.New(rand.NewSource(42))
	components := []string{
		"src", "main", "cmd", "tools", "utils", "common", "query", "process", "update",
		"run", "build", "assets", "frontend", "backend", "worker",
	}
	libs := make([]core.BuildLabel, 0, numPackages)
	addTarget := func(pkg *core.Package, name string) *core.BuildTarget {
		t := core.NewBuildTarget(core.BuildLabel{PackageName: pkg.Name, Name: name})
		t.Command = fmt.Sprintf("$TOOLS_GO tool compile -o $OUT -p %s $SRCS", pkg.Name)
		t.AddOutput(name + ".a")
		t.AddLabel("go")
		t.AddTool(core.SystemPathLabel{Name: "go", Path: state.Config.Path()})
		t.Visibility = []core.BuildLabel{core.WholeGraph[0]}
		state.Graph.AddTarget(t)
		pkg.AddTarget(t)
		return t
	}
	for i := 0; i < numPackages; i++ {
		name := fmt.Sprintf("%s/%s/pkg%d", components[r.Intn(len(components))], components[r.Intn(len(components))], i)
		pkg := core.NewPackage(name)
		state.Graph.AddPackage(pkg)
		base := components[r.Intn(len(components))]

		srcs := addTarget(pkg, "_"+base+"#srcs")
		for j := 0; j < 1+r.Intn(10); j++ {
			srcs.AddSource(core.FileLabel{File: fmt.Sprintf("%s_%d.go", base, j), Package: name})
		}
		asm := addTarget(pkg, "_"+base+"#asm")
		asm.AddDependency(srcs.Label)
		lib := addTarget(pkg, base)
		lib.AddSource(srcs.Label)
		lib.AddDependency(srcs.Label)
		lib.AddDependency(asm.Label)
		for j := 0; j < len(libs) && j < 1+r.Intn(8); j++ {
			lib.AddDependency(libs[r.Intn(len(libs))])
		}
		libs = append(libs, lib.Label)

		test := addTarget(pkg, base+"_test")
		test.AddSource(core.FileLabel{File: base + "_test.go", Package: name})
		test.AddDependency(lib.Label)
		test.Test = &core.TestFields{Command: "$TEST"}

		bin := addTarget(pkg, base+"_bin")
		bin.AddSource(core.FileLabel{File: "main.go", Package: name})
		bin.AddDependency(lib.Label)
		bin.IsBinary = true
	}
	return state
}
