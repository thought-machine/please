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

// SmallShardCount is a shard count useful for relatively small maps.
const SmallShardCount = 4

// A Map is the top-level map type. All functions on it are threadsafe.
// It should be constructed via New() rather than creating an instance directly.
type Map[K comparable, V any] struct {
	shards []shard[K, V]
	mask   uint64
	hasher func(K) uint64
}

// New creates a new Map using the given hasher to hash items in it.
// The shard count must be a power of 2; it will panic if not.
// Higher shard counts will improve concurrency but consume more memory.
// The DefaultShardCount of 256 is reasonable for a large map.
func New[K comparable, V any](shardCount uint64, hasher func(K) uint64) *Map[K, V] {
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

// Add adds the new item to the map.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (m *Map[K, V]) Add(key K, val V) bool {
	return m.shards[m.hasher(key)&m.mask].Set(key, val, false)
}

// AddOrGet either adds a new item (if the key doesn't exist, calling the given function to create it) or gets the existing one.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (m *Map[K, V]) AddOrGet(key K, f func() V) (V, bool) {
	return m.shards[m.hasher(key)&m.mask].LazySet(key, f)
}

// Set is the equivalent of `map[key] = val`.
// It always overwrites any key that existed before.
func (m *Map[K, V]) Set(key K, val V) {
	m.shards[m.hasher(key)&m.mask].Set(key, val, true)
}

// Get returns the value corresponding to the given key, or its zero value if the key doesn't exist in the map.
func (m *Map[K, V]) Get(key K) V {
	v, _, _ := m.shards[m.hasher(key)&m.mask].Get(key)
	return v
}

func (m *Map[K, V]) Contains(key K) bool {
	return m.shards[m.hasher(key)&m.mask].Contains(key)
}

// GetOrWait returns the value or, if the key isn't present, a channel that it can be waited
// on for. The caller will need to call Get again after the channel closes.
// If the channel is non-nil, then val will exist in the map; otherwise it will have its zero value.
// The third return value is true if this is the first call that is awaiting this key.
// It's always false if the key exists.
func (m *Map[K, V]) GetOrWait(key K) (val V, wait <-chan struct{}, first bool) {
	return m.shards[m.hasher(key)&m.mask].Get(key)
}

// Values returns a slice of all the current values in the map.
// No particular consistency guarantees are made.
func (m *Map[K, V]) Values() []V {
	ret := []V{}
	for i := 0; i < len(m.shards); i++ {
		ret = append(ret, m.shards[i].Values()...)
	}
	return ret
}

// Range calls f for each key-value pair in the map.
// No particular consistency guarantees are made during iteration.
func (m *Map[K, V]) Range(f func(key K, val V)) {
	for i := 0; i < len(m.shards); i++ {
		m.shards[i].Range(f)
	}
}

// An awaitableValue represents a value in the map & an awaitable channel for it to exist.
type awaitableValue[V any] struct {
	Val  V
	Wait chan struct{}
}

// A shard is one of the individual shards of a map.
type shard[K comparable, V any] struct {
	m map[K]awaitableValue[V]
	l sync.RWMutex
}

// Set is the equivalent of `map[key] = val`.
// It returns true if the item was inserted, false if it was not
// (because an existing one was found and overwrite was false).
func (s *shard[K, V]) Set(key K, val V, overwrite bool) bool {
	s.l.Lock()
	defer s.l.Unlock()
	if existing, present := s.m[key]; present {
		if existing.Wait == nil {
			if !overwrite {
				return false // already added
			}
			s.m[key] = awaitableValue[V]{Val: val}
			return true
		}
		// Hasn't been added, but something is waiting for it to be.
		s.m[key] = awaitableValue[V]{Val: val}
		close(existing.Wait)
		existing.Wait = nil
		return true
	}
	s.m[key] = awaitableValue[V]{Val: val}
	return true
}

// LazySet is like Set but calls the given function to construct the object only if needed.
// It also returns the value that is now set in the map (whether overwritten or not).
func (s *shard[K, V]) LazySet(key K, f func() V) (V, bool) {
	s.l.Lock()
	defer s.l.Unlock()
	if existing, present := s.m[key]; present {
		if existing.Wait == nil {
			return existing.Val, false // already added
		}
		// Hasn't been added, but something is waiting for it to be.
		v := f()
		s.m[key] = awaitableValue[V]{Val: v}
		close(existing.Wait)
		existing.Wait = nil
		return v, true
	}
	v := f()
	s.m[key] = awaitableValue[V]{Val: v}
	return v, true
}

// Get returns the value for a key or, if not present, a channel that it can be waited
// on for.
// Exactly one of the target or channel will be returned.
// The third value is true if it is the first call that is waiting on this value.
func (s *shard[K, V]) Get(key K) (val V, wait <-chan struct{}, first bool) {
	s.l.RLock()
	if v, ok := s.m[key]; ok {
		s.l.RUnlock()
		return v.Val, v.Wait, false
	}
	s.l.RUnlock()

	s.l.Lock()
	defer s.l.Unlock()
	if v, ok := s.m[key]; ok {
		return v.Val, v.Wait, false
	}
	ch := make(chan struct{})
	s.m[key] = awaitableValue[V]{Wait: ch}
	wait = ch
	first = true
	return
}

// Values returns a copy of all the targets currently in the map.
func (s *shard[K, V]) Values() []V {
	s.l.RLock()
	defer s.l.RUnlock()
	ret := make([]V, 0, len(s.m))
	for _, v := range s.m {
		if v.Wait == nil {
			ret = append(ret, v.Val)
		}
	}
	return ret
}

func (s *shard[K, V]) Contains(key K) bool {
	s.l.RLock()
	defer s.l.RUnlock()

	_, ok := s.m[key]
	return ok
}

// Range calls f for each key-value pair in this shard.
func (s *shard[K, V]) Range(f func(key K, val V)) {
	s.l.RLock()
	defer s.l.RUnlock()
	for k, v := range s.m {
		if v.Wait == nil { // Only include completed values
			f(k, v.Val)
		}
	}
}
