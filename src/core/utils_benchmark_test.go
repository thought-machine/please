package core

import (
	"fmt"
	"testing"
)

func BenchmarkIterInputsControl(b *testing.B) {
	state := NewDefaultBuildState()
	target := NewBuildTarget(NewBuildLabel("src/foo", "foo_lib"))
	state.Graph.AddTarget(target)

	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range IterInputs(state, state.Graph, target, true, false) {
		}
	}
}

func BenchmarkIterInputsSimple(b *testing.B) {
	state := NewDefaultBuildState()

	target := NewBuildTarget(NewBuildLabel("src/foo", "foo_lib"))
	target.NeedsTransitiveDependencies = true
	for i := 0; i < 100; i++ {
		dep := NewBuildTarget(NewBuildLabel("src/bar", fmt.Sprintf("name_%d", i)))
		for j := 0; j < 200; j++ {
			dep.AddOutput(fmt.Sprintf("name_%v_file_%v", i, j))
		}
		state.Graph.AddTarget(dep)
		target.AddDependency(dep.Label)
		target.resolveDependency(target.Label, dep)
	}

	for i := 0; i < 25; i++ {
		target.AddSource(FileLabel{
			File:    fmt.Sprintf("src_%v", i),
			Package: "src/foo",
		})
	}

	state.Graph.AddTarget(target)
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range IterInputs(state, state.Graph, target, true, false) {
		}
	}
}

func BenchmarkIterInputsNamedSources(b *testing.B) {
	state := NewDefaultBuildState()

	target := NewBuildTarget(NewBuildLabel("src/foo", "foo_lib"))
	target.NeedsTransitiveDependencies = true
	for i := 0; i < 100; i++ {
		dep := NewBuildTarget(NewBuildLabel("src/bar", fmt.Sprintf("name_%d", i)))
		for j := 0; j < 200; j++ {
			dep.AddOutput(fmt.Sprintf("name_%v_file_%v", i, j))
		}
		state.Graph.AddTarget(dep)
		target.AddDependency(dep.Label)
		target.resolveDependency(target.Label, dep)
	}

	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			target.AddNamedSource(fmt.Sprintf("srcs_%v", i), FileLabel{
				File:    fmt.Sprintf("src_%v_%v", i, j),
				Package: "src/foo",
			})
		}
	}

	state.Graph.AddTarget(target)
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range IterInputs(state, state.Graph, target, true, false) {
		}
	}
}
