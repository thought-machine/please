package cmap

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

// AddOrGet either adds a new item (if the key doesn't exist) or gets the existing one.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (m *ErrMap[K, V]) AddOrGet(key K, f func() V) (V, bool, error) {
	v, present := m.m.AddOrGet(key, func() errV[V] { return errV[V]{Val: f()} })
	return v.Val, present, v.Err
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
func (m *ErrMap[K, V]) GetOrSet(key K, f func() (V, error)) (V, error) {
	v, wait, first := m.m.GetOrWait(key)
	if v.Err != nil {
		return v.Val, v.Err
	} else if first {
		val, err := f()
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

// Range calls f for each key-value pair in the map.
// No particular consistency guarantees are made during iteration.
func (m *ErrMap[K, V]) Range(f func(key K, val V)) {
	m.m.Range(func(key K, val errV[V]) {
		if val.Err != nil {
			return // skip errors
		}
		f(key, val.Val)
	})
}
