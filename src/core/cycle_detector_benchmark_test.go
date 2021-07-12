package core

import (
	"fmt"
	"testing"
	"time"
)

func buildTree(levels, width int, from BuildLabel) []dependencyLink {
	if levels == 0 {
		return []dependencyLink{}
	}

	var links []dependencyLink
	for i := 0; i < width; i++ {
		to := ParseBuildLabel(fmt.Sprintf("%v_%v", from, i), "")
		links = append(links, dependencyLink{from: from, to: to})
		links = append(links, buildTree(levels-1, width, to)...)
	}
	return links
}

func BenchmarkCycles(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		benchmarkCycleDetector(b, 2)
	})

	b.Run("Medium", func(b *testing.B) {
		benchmarkCycleDetector(b, 4)
	})

	b.Run("Large", func(b *testing.B) {
		benchmarkCycleDetector(b, 6)
	})
}

func benchmarkCycleDetector(b *testing.B, graphLevels int) {
	from := ParseBuildLabel("//src:root", "")
	// Generates a tree of almost 20k targets for large (level 6)
	links := buildTree(graphLevels, 5, from)

	cycleDetectors := make([]*cycleDetector, b.N)
	for n := 0; n < b.N; n++ {
		cycleDetectors[n] = newCycleDetector()
	}

	b.ReportAllocs()
	b.ResetTimer()
	start := time.Now()

	for n := 0; n < b.N; n++ {
		cd := cycleDetectors[n]
		for _, link := range links {
			if err := cd.addDep(link); err != nil {
				panic(err)
			}
		}
	}

	duration := time.Now().Sub(start) / time.Millisecond
	linksPerMS := float64(len(links)*b.N) / float64(duration)
	b.ReportMetric(linksPerMS, "links/ms")
}
