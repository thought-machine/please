package cmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type hashInts struct{}

func (h hashInts) Hash(k int) uint32 { return uint32(k) }

func TestMap(t *testing.T) {
	m := New[int, int, hashInts](DefaultShardCount)
	assert.True(t, m.Set(5, 7))
	assert.True(t, m.Set(7, 5))
}

func BenchmarkMapInserts(b *testing.B) {
	m := New[int, int, hashInts](DefaultShardCount)
	for i := 0; i < b.N; i++ {
		m.Set(i, i)
	}
}
