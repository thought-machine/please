package core

import (
	"math/rand"
	"strings"
	"testing"
)

// Construct one set of targets that are shared between all benchmarks, so we don't have to
// rebuild them for every benchmark (which messes with timing quite a lot)
var targets []*BuildTarget

var graph *BuildGraph

const numTargets = 1 << 20

const targetIndexMask = numTargets - 1

func init() {
	log.Notice("Initialising...")
	targets = createTargets(numTargets)
	graph = NewGraph()
	for _, target := range targets {
		graph.AddTarget(target)
	}
	log.Notice("Initialised targets...")
}

func BenchmarkAddingTargets(b *testing.B) {
	graph := NewGraph()
	b.ResetTimer()  // Don't benchmark graph creation
	b.ReportAllocs()
	for _, target := range targets[:b.N] {
		graph.AddTarget(target)
	}
}

func BenchmarkTargetLookup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		graph.TargetOrDie(targets[i & targetIndexMask].Label)
	}
}

// BenchmarkWaitForTargetFast benchmarks the best case of calling WaitForTarget,
// where the targets already exist, so it should perform identically to BenchmarkTargetLookup.
func BenchmarkWaitForTargetFast(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		graph.WaitForTarget(targets[i & targetIndexMask].Label)
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
