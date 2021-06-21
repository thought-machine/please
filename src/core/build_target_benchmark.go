package core

import (
	"testing"
)

// Use this to prevent compiler optimisations from removing anything.
var result []BuildLabel //nolint:unused

func BenchmarkProvideFor(b *testing.B) {
	target1 := NewBuildTarget(BuildLabel{PackageName: "src/core", Name: "target1"})
	target2 := NewBuildTarget(BuildLabel{PackageName: "src/core", Name: "target2"})
	target3 := NewBuildTarget(BuildLabel{PackageName: "src/core", Name: "target3"})
	target4 := NewBuildTarget(BuildLabel{PackageName: "src/core", Name: "target4"})

	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result = target1.provideFor(target2)
		}
	})

	target1.Requires = []string{"go"}
	target2.AddProvide("py", target3.Label)
	b.Run("NoMatch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result = target2.provideFor(target1)
		}
	})

	target2.AddProvide("go", target3.Label)
	target2.AddProvide("go_src", target4.Label)
	b.Run("OneMatch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result = target2.provideFor(target1)
		}
	})

	target1.Requires = []string{"go", "go_src"}
	b.Run("TwoMatches", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result = target2.provideFor(target1)
		}
	})

	target1.AddDatum(target2.Label)
	b.Run("IsData", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result = target2.provideFor(target1)
		}
	})
}
