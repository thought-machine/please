package cmap

import (
	"context"
	"fmt"
	"iter"
	"runtime/debug"
)

// A Limiter is the interface that we use to release/acquire workers while waiting.
type Limiter interface {
	Acquire()
	Release()
}

// NewErrMap returns a map that extends Map with an error type, which callers can also wait on
// and receive if something goes wrong.
func NewErrMap[K comparable, V any](shardCount uint64, hasher func(K) uint64, limiter Limiter) *ErrMap[K, V] {
	return &ErrMap[K, V]{
		m: New[K, errV[V]](shardCount, hasher),
		l: limiter,
	}
}

type errV[V any] struct {
	Err error
	Val V
}

// An ErrMap extends Map with returned errors as a first-class concept
type ErrMap[K comparable, V any] struct {
	m *Map[K, errV[V]]
	l Limiter
}

// Add adds the new item to the map.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (m *ErrMap[K, V]) Add(key K, val V) bool {
	return m.m.Add(key, errV[V]{Val: val})
}

// Set is the equivalent of `map[key] = val`.
// It always overwrites any key that existed before.
func (m *ErrMap[K, V]) Set(key K, val V) {
	m.m.Set(key, errV[V]{Val: val})
}

// SetError overwrites the key with the given error.
func (m *ErrMap[K, V]) SetError(key K, err error) {
	m.m.Set(key, errV[V]{Err: err})
}

// Get returns the value corresponding to the given key, or its zero value if the key doesn't exist in the map.
// If an error has been set for the key, that will be returned.
func (m *ErrMap[K, V]) Get(key K) (V, error) {
	v := m.m.Get(key)
	return v.Val, v.Err
}

// GetOrSet returns the value if set, or an error if one has been set.
// If nothing has been set for the key, it runs the given function to generate the value and then sets it.
// If the called function panics, the panic will be recovered and treated as though it returned an error.
func (m *ErrMap[K, V]) GetOrSet(key K, f func() (V, error)) (val V, err error) {
	v, wait, first := m.m.GetOrWait(key)
	if v.Err != nil {
		return v.Val, v.Err
	} else if first {
		defer func() {
			if r := recover(); r != nil {
				err = panicToErr(r)
				m.m.Set(key, errV[V]{Err: err})
			}
		}()
		val, err = f()
		m.m.Set(key, errV[V]{Val: val, Err: err})
		return val, err
	} else if wait != nil {
		if m.l != nil {
			// Release the limiter for the duration we're waiting
			m.l.Release()
			defer m.l.Acquire()
		}
		<-wait
		return m.Get(key)
	}
	return v.Val, v.Err
}

// panicToErr converts a recovered panic value into an error, retaining its type if it already is an error.
func panicToErr(r any) error {
	if e, ok := r.(error); ok {
		return e
	}
	// Maintain the stack; if something panicked that wasn't an error we will probably want it for debugging.
	return fmt.Errorf("%v\n%s", r, debug.Stack())
}

// GetOrSetCtx is like GetOrSet but accepts a context that can be cancelled.
// If the called function panics, the panic will be recovered and treated as though it returned an error.
func (m *ErrMap[K, V]) GetOrSetCtx(ctx context.Context, key K, f func() (V, error)) (val V, err error) {
	v, wait, first := m.m.GetOrWait(key)
	if v.Err != nil {
		return v.Val, v.Err
	} else if first {
		defer func() {
			if r := recover(); r != nil {
				err = panicToErr(r)
				m.m.Set(key, errV[V]{Err: err})
			}
		}()
		val, err = f()
		m.m.Set(key, errV[V]{Val: val, Err: err})
		return val, err
	} else if wait != nil {
		if m.l != nil {
			// Release the limiter for the duration we're waiting
			m.l.Release()
			defer m.l.Acquire()
		}
		select {
		case <-wait:
			return m.Get(key)
		case <-ctx.Done():
			var v V
			return v, ctx.Err()
		}
	}
	return v.Val, v.Err
}

// GetOrWait returns the value for a key, or an error if one has been recorded for it.
// If neither is set yet, it returns a channel that is closed once the key is populated with a value or
// an error, and an indication of whether the caller is the first to request it.
func (m *ErrMap[K, V]) GetOrWait(key K) (V, <-chan struct{}, bool, error) {
	v, wait, first := m.m.GetOrWait(key)
	return v.Val, wait, first, v.Err
}

// Items returns an iterator over all key / value pairs in the map, which have not errored.
// They are returned in no particular order.
// You should not mutate the map while calling this as it may deadlock.
func (m *ErrMap[K, V]) Items() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range m.m.Items() {
			if v.Err == nil {
				if !yield(k, v.Val) {
					return
				}
			}
		}
	}
}

// Values returns an iterator over all value / error pairs in the map.
// They are returned in no particular order.
// You should not mutate the map while calling this as it may deadlock.
func (m *ErrMap[K, V]) Values() iter.Seq2[V, error] {
	return func(yield func(V, error) bool) {
		for _, v := range m.m.Items() {
			if !yield(v.Val, v.Err) {
				return
			}
		}
	}
}
