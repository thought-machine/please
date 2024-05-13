package cmap

// NewErrMap returns a map that extends Map with an error type, which callers can also wait on
// and receive if something goes wrong.
func NewErrMap[K comparable, V any](shardCount uint64, hasher func(K) uint64) *ErrMap[K, V] {
	return &ErrMap[K, V]{
		m: New[K, errV[V]](shardCount, hasher),
	}
}

type errV[V any] struct {
	Err error
	Val V
}

// An ErrMap extends Map with returned errors as a first-class concept
type ErrMap[K comparable, V any] struct {
	m *Map[K, errV[V]]
}

// Add adds the new item to the map.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (m *ErrMap[K, V]) Add(key K, val V) bool {
	return m.m.Add(key, errV[V]{Val: val})
}

// AddOrGet either adds a new item (if the key doesn't exist) or gets the existing one.
// It returns true if the item was inserted, false if it already existed (in which case it won't be inserted)
func (m *ErrMap[K, V]) AddOrGet(key K, val V) (V, bool, error) {
	v, present := m.m.AddOrGet(key, errV[V]{Val: val})
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

// GetOrWait returns the value or, if the key isn't present, a channel that it can be waited
// on for. The caller will need to call Get again after the channel closes.
// If the channel is non-nil, then val will exist in the map; otherwise it will have its zero value.
// The third return value is true if this is the first call that is awaiting this key.
// It's always false if the key exists.
func (m *ErrMap[K, V]) GetOrWait(key K) (val V, wait <-chan struct{}, first bool, err error) {
	v, wait, first := m.m.GetOrWait(key)
	return v.Val, wait, first, v.Err
}
