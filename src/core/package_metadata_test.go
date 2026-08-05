package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashBuildStatementDistribution(t *testing.T) {
	nShards := 4
	shards := make(map[uint64]int, nShards)
	totalStatements := 1000

	for i := 0; i < totalStatements; i++ {
		start := i * 4                    // Simulate predictable start
		end := start + 31*(i%4) + (i % 7) // Varied statement lengths
		stmt := BuildStatement{Start: start, End: end}

		h := hashBuildStatement(stmt)
		shard := h & (uint64(nShards) - 1)
		shards[shard]++
	}

	// We expect a reasonably balanced distribution across all 4.
	// With 1000 statements, each shard should ideally have ~250.
	// We'll assert that every shard gets at least 15% of the total statements
	// and no shard gets more than 35%, allowing room for 10% variance.
	optimal := int(totalStatements / nShards)
	minExpected := int(float64(optimal) * 0.9)
	maxExpected := int(float64(optimal) * 1.1)

	assert.Len(t, shards, 4, "All 4 shards must be populated")
	for shard, count := range shards {
		assert.GreaterOrEqual(t, count, minExpected, "Shard %d has poor representation: %d", shard, count)
		assert.LessOrEqual(t, count, maxExpected, "Shard %d is overly congested: %d", shard, count)
		fmt.Println(shard, count)
	}
}
