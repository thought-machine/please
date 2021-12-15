// Package cmap contains a thread-safe concurrent awaitable map.
// It is optimised for large maps (e.g. tens of thousands of entries) in highly
// contended environments; for smaller maps another implementation may do better.
//
// Only slightly ad-hoc testing has shown it to outperform sync.Map for our uses
// due to less contention. It is also specifically useful in cases where a caller
// wants to be able to await items entering the map (and not having to poll it to
// find out when another goroutine may insert them).
package cmap

import (
	"fmt"
	"sync"
)

// DefaultShardCount is a reasonable default shard count for large maps.
const DefaultShardCount = 1 << 8

// A Map is the top-level map type. All functions on it are threadsafe.
// It should be constructed via New() rather than creating an instance directly.
type Map[K comparable, V any] struct {
	shards []shard[K, V]
	hasher func(K) uint32
	mask   uint32
}

// New creates a new Map using the given hasher to hash items in it.
// The shard count must be a power of 2; it will panic if not.
// Higher shard counts will improve concurrency but consume more memory.
// The DefaultShardCount of 256 is reasonable for a large map.
func New[K comparable, V any](shardCount uint32, hasher func(K) uint32) *Map[K, V] {
	mask := shardCount - 1
	if (shardCount & mask) != 0 {
		panic(fmt.Sprintf("Shard count %d is not a power of 2", shardCount))
	}
	m := &Map[K, V]{
		shards: make([]shard[K, V], shardCount),
		mask:   mask,
		hasher: hasher,
	}
	for i := range m.shards {
		m.shards[i].m = map[K]awaitableValue[V]{}
	}
	return m
}

// Set is the equivalent of `map[key] = val`.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (m *Map[K, V]) Set(key K, val V) bool {
	return m.shards[m.hasher(key)&m.mask].Set(key, val)
}

// Get returns the value or, if the key isn't present, a channel that it can be waited
// on for. The caller will need to call Get again after the channel closes.
// Exactly one of the value or channel will be returned.
func (m *Map[K, V]) Get(key K) (val V, wait <-chan struct{}) {
	return m.shards[m.hasher(key)&m.mask].Get(key)
}

// Values returns a slice of all the current values in the map.
// No particular consistency guarantees are made.
func (m *Map[K, V]) Values() []V {
	ret := []V{}
	for _, shard := range m.shards {
		ret = append(ret, shard.Values()...)
	}
	return ret
}

// An awaitableValue represents a value in the map & an awaitable channel for it to exist.
type awaitableValue[V any] struct {
	Val  V
	Wait chan struct{}
}

// A shard is one of the individual shards of a map.
type shard[K comparable, V any] struct {
	m map[K]awaitableValue[V]
	l sync.Mutex
}

// Set is the equivalent of `map[key] = val`.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (s *shard[K, V]) Set(key K, val V) bool {
	s.l.Lock()
	defer s.l.Unlock()
	if existing, present := s.m[key]; present {
		if existing.Wait == nil {
			return false // already added
		}
		// Hasn't been added, but something is waiting for it to be.
		s.m[key] = awaitableValue[V]{Val: val}
		if existing.Wait != nil {
			close(existing.Wait)
			existing.Wait = nil
		}
		return true
	}
	s.m[key] = awaitableValue[V]{Val: val}
	return true
}

// Get returns the value for a key or, if not present, a channel that it can be waited
// on for.
// Exactly one of the target or channel will be returned.
func (s *shard[K, V]) Get(key K) (val V, wait <-chan struct{}) {
	s.l.Lock()
	defer s.l.Unlock()
	if v, ok := s.m[key]; ok {
		return v.Val, v.Wait
	}
	ch := make(chan struct{})
	s.m[key] = awaitableValue[V]{Wait: ch}
	wait = ch
	return
}

// Values returns a copy of all the targets currently in the map.
func (s *shard[K, V]) Values() []V {
	s.l.Lock()
	defer s.l.Unlock()
	ret := make([]V, 0, len(s.m))
	for _, v := range s.m {
		if v.Wait == nil {
			ret = append(ret, v.Val)
		}
	}
	return ret
}
