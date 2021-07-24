package core

import (
	"math/rand"
	"strings"
	"testing"
)

func BenchmarkAddingTargets(b *testing.B) {
	targets := createTargets(b.N)
	graph := NewGraph()
	b.ResetTimer()  // Don't benchmark target creation
	b.ReportAllocs()
	for _, target := range targets {
		graph.AddTarget(target)
	}
}

func BenchmarkTargetLookup(b *testing.B) {
	targets := createTargets(b.N)
	graph := NewGraph()
	for _, target := range targets {
		graph.AddTarget(target)
	}
	b.ResetTimer()  // Don't benchmark graph creation
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Adding this multiplier sucks a bit, but without it the benchmark takes ~1min to
		// converge; with it it's about a second.
		for j := 0; j < 100; j++ {
			graph.TargetOrDie(targets[i].Label)
		}
	}
}

func BenchmarkWaitForTarget(b *testing.B) {
	targets := createTargets(b.N)
	graph := NewGraph()
	for _, target := range targets {
		graph.AddTarget(target)
	}
	b.ResetTimer()  // Don't benchmark graph creation
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Adding this multiplier sucks a bit, but without it the benchmark takes ~1min to
		// converge; with it it's about a second.
		for j := 0; j < 100; j++ {
			graph.WaitForTarget(targets[i].Label)
		}
	}
}

// createTargets creates n randomly named targets.
func createTargets(n int) []*BuildTarget {
	targets := make([]*BuildTarget, n)
	seen := map[BuildLabel]bool{}

	rand.Seed(42)  // Make sure it does the same thing every time.

	components := []string{
		"src", "main", "cmd", "tools", "utils", "common", "query", "process", "update",
		"run", "build", "assets", "frontend", "backend", "worker",
	}
	component := func() string { return components[rand.Intn(len(components))] }
	label := func() BuildLabel {
		for i := 0; i < 1000; i++ {
			parts := []string{}
			for j := 0; j < rand.Intn(7); j++ {
				parts = append(parts, component())
			}
			l := BuildLabel{
				PackageName: strings.Join(parts, "/"),
				Name:        component(),
			}
			if !seen[l] {
				seen[l] = true
				return l
			}
		}
		panic("Couldn't generate a unique name after 1000 attempts")
	}
	for i := 0; i < n; i++ {
		targets[i] = NewBuildTarget(label())
	}
	return targets
}
