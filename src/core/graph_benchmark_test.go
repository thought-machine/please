package core

import (
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
)

func BenchmarkAddingTargets(b *testing.B) {
	targets := createTargets(b.N)
	graph := NewGraph()
	b.ResetTimer() // Don't benchmark graph creation
	b.ReportAllocs()
	for _, target := range targets[:b.N] {
		graph.AddTarget(target)
	}
}

func BenchmarkTargetLookup(b *testing.B) {
	// Do all the setup in an initial step. This is relatively slow and we don't want it to
	// count towards the benchmarks themselves.
	const numTargets = 1 << 20
	const targetIndexMask = numTargets - 1
	targets := createTargets(numTargets)
	graph := NewGraph()
	for _, target := range targets {
		graph.AddTarget(target)
	}
	b.ReportAllocs()

	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			graph.TargetOrDie(targets[i&targetIndexMask].Label)
		}
	})

	// This benchmarks the best case of calling WaitForTarget, where the targets already exist,
	// so it should perform identically to Simple above.
	b.Run("WaitForTargetFast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			graph.WaitForTarget(targets[i&targetIndexMask].Label)
		}
	})
}

// BenchmarkWaitForTargetSlow is a more complex benchmark that tests targets being added as they are
// being waited on.
func BenchmarkWaitForTargetSlow(b *testing.B) {
	const parallelism = 8
	var wg sync.WaitGroup
	wg.Add(parallelism * 2)

	targets := createTargets(b.N)
	graph := NewGraph()
	b.ResetTimer()
	b.ReportAllocs()

	addTargets := func(start int) {
		for i := start; i < b.N; i += parallelism {
			graph.AddTarget(targets[i])
		}
		wg.Done()
	}

	lookupTargets := func() {
		for _, target := range targets {
			graph.WaitForTarget(target.Label)
		}
		wg.Done()
	}

	for i := 0; i < parallelism; i++ {
		go addTargets(i)
		go lookupTargets()
	}
	wg.Wait()
}

// createTargets creates n randomly named targets.
func createTargets(n int) []*BuildTarget {
	targets := make([]*BuildTarget, n)
	seen := map[BuildLabel]bool{}

	r := rand.New(rand.NewChaCha8([32]byte{})) // Make sure it does the same thing every time.

	components := []string{
		"src", "main", "cmd", "tools", "utils", "common", "query", "process", "update",
		"run", "build", "assets", "frontend", "backend", "worker",
	}
	component := func() string { return components[r.IntN(len(components))] }
	label := func() BuildLabel {
		for i := 0; i < 1000; i++ {
			parts := []string{}
			for j := 0; j < r.IntN(7); j++ {
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
