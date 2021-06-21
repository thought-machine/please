package benchmark

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
	"testing"
)

func BenchmarkIterInputsSimple(b *testing.B) {
	state := core.NewDefaultBuildState()

	target := core.NewBuildTarget(core.NewBuildLabel("src/foo", "foo_lib"))
	for i := 0; i < 100; i++ {
		dep := core.NewBuildTarget(core.NewBuildLabel("src/bar", fmt.Sprintf("name_%d", i)))
		for j := 0; j < 200; j++ {
			dep.AddOutput(fmt.Sprintf("name_%v_file_%v", i, j))
		}
		state.Graph.AddTarget(dep)
		target.AddDependency(dep.Label)
	}

	for i := 0; i < 25; i++ {
		target.AddSource(core.FileLabel{
			File:    fmt.Sprintf("src_%v", i),
			Package: "src/foo",
		})
	}

	state.Graph.AddTarget(target)
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range core.IterInputs(state.Graph, target, true, false) {}
	}
}


func BenchmarkIterInputsNamedSources(b *testing.B) {
	state := core.NewDefaultBuildState()

	target := core.NewBuildTarget(core.NewBuildLabel("src/foo", "foo_lib"))
	for i := 0; i < 100; i++ {
		dep := core.NewBuildTarget(core.NewBuildLabel("src/bar", fmt.Sprintf("name_%d", i)))
		for j := 0; j < 200; j++ {
			dep.AddOutput(fmt.Sprintf("name_%v_file_%v", i, j))
		}
		state.Graph.AddTarget(dep)
		target.AddDependency(dep.Label)
	}

	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			target.AddNamedSource(fmt.Sprintf("srcs_%v", i), core.FileLabel{
				File:    fmt.Sprintf("src_%v_%v", i, j),
				Package: "src/foo",
			})
		}
	}

	state.Graph.AddTarget(target)
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range core.IterInputs(state.Graph, target, true, false) {}
	}
}