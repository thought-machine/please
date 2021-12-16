package cmap

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func hashInts(k int) uint32 { return uint32(k) }

func TestMap(t *testing.T) {
	m := New[int, int](DefaultShardCount, hashInts)
	assert.True(t, m.Add(5, 7))
	assert.True(t, m.Add(7, 5))
	assert.Equal(t, 7, m.Get(5))
	assert.Equal(t, 5, m.Get(7))
	vals := m.Values()
	// Order isn't guaranteed so we must sort it now.
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	assert.Equal(t, []int{5, 7}, vals)
}

func TestWait(t *testing.T) {
	m := New[int, int](DefaultShardCount, hashInts)
	v, ch, first := m.GetOrWait(5)
	assert.Equal(t, 0, v) // Should be the zero value
	assert.True(t, first) // We're the first to request it
	go func() {
		m.Set(5, 7)
	}()
	<-ch
	v, ch, first = m.GetOrWait(5)
	assert.Nil(t, ch)
	assert.Equal(t, 7, v)
	assert.False(t, first)
}

func TestReAdd(t *testing.T) {
	m := New[int, int](DefaultShardCount, hashInts)
	assert.True(t, m.Add(5, 7))
	assert.False(t, m.Add(5, 7))
	v, ch, first := m.GetOrWait(5)
	assert.Nil(t, ch)
	assert.Equal(t, 7, v)
	assert.False(t, first)
	m.Set(5, 8)
	v, ch, first = m.GetOrWait(5)
	assert.Nil(t, ch)
	assert.Equal(t, 8, v)
	assert.False(t, first)
}

func TestShardCount(t *testing.T) {
	New[int, int](4, hashInts)
	assert.Panics(t, func() {
		New[int, int](3, hashInts)
	})
}

func BenchmarkMapInserts(b *testing.B) {
	m := New[int, int](DefaultShardCount, hashInts)
	for i := 0; i < b.N; i++ {
		m.Set(i, i)
	}
}
