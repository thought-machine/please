package cmap

import (
	"fmt"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func hashInts(k int) uint64 {
	return XXHash(strconv.Itoa(k))
}

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

// TestWait covers the awaiting primitive directly; it's only reachable through ErrMap now,
// but it's the bit with the interesting concurrency so it's worth pinning down here.
func TestWait(t *testing.T) {
	m := New[int, int](DefaultShardCount, hashInts)
	v, ch, first := m.getOrWait(5)
	assert.Equal(t, 0, v) // Should be the zero value
	assert.True(t, first) // We're the first to request it
	go func() {
		m.Set(5, 7)
	}()
	<-ch
	v, ch, first = m.getOrWait(5)
	assert.Nil(t, ch)
	assert.Equal(t, 7, v)
	assert.False(t, first)
}

func TestGetDoesntInsert(t *testing.T) {
	m := New[int, int](DefaultShardCount, hashInts)
	assert.Equal(t, 0, m.Get(5))
	// A failed lookup must not leave an entry behind; anything that later tries to set this key
	// would find something already waiting on it and never get to do the work.
	assert.False(t, m.Contains(5))
}

func TestReAdd(t *testing.T) {
	m := New[int, int](DefaultShardCount, hashInts)
	assert.True(t, m.Add(5, 7))
	assert.False(t, m.Add(5, 7))
	assert.Equal(t, 7, m.Get(5))
	m.Set(5, 8)
	assert.Equal(t, 8, m.Get(5))
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

func TestResize(t *testing.T) {
	for n := 10; n <= 1000; n *= 10 {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			m := New[int, int](1, hashInts)
			for i := 0; i < n; i++ {
				m.Set(i, i)
			}
			for i := 0; i < n; i++ {
				assert.Equal(t, i, m.Get(i), "Key %d appears to be not set or set incorrectly", i)
			}
		})
	}
}

func BenchmarkMapInsertsAndGets(b *testing.B) {
	// Attempts to mimic a vaguely realistic blend of writes and (more) reads.
	m := New[int, int](DefaultShardCount, hashInts)
	var wg, rg errgroup.Group
	wg.SetLimit(3)
	rg.SetLimit(12)
	for i := 0; i < b.N; i++ {
		x := i
		for j := 0; j < 10; j++ {
			wg.Go(func() error {
				for k := 0; k < 1000; k++ {
					m.Set(x, x)
				}
				return nil
			})
		}
		for j := 0; j < 100; j++ {
			rg.Go(func() error {
				for k := 0; k < 1000; k++ {
					if y := m.Get(x); y != x && y != 0 {
						return fmt.Errorf("incorrect result, was %d, should be %d", y, x)
					}
				}
				return nil
			})
		}
	}
	assert.NoError(b, wg.Wait())
	assert.NoError(b, rg.Wait())
}
