package asp

import (
	"sync"
)

// A subincludeResult holds the value in a subincludeMap.
type subincludeResult struct {
	Globals pyDict
	Wait    chan struct{}
}

// A subincludeMap stores the results of subinclude calls in a way that they
// can be waited on.
type subincludeMap struct{
	m map[string]subincludeResult
	l sync.Mutex
}

// Set is the equivalent of `map[key] = val`.
func (m *subincludeMap) Set(key string, val pyDict) {
	m.l.Lock()
	defer m.l.Unlock()
	if existing, present := m.m[key]; present {
		// Hasn't been added, but something is waiting for it to be.
		m.m[key] = subincludeResult{Globals: val}
		if existing.Wait != nil {
			close(existing.Wait)
		}
		return
	}
	m.m[key] = subincludeResult{Globals: val}
}

// Get returns the subinclude globals or, if it hasn't been parsed yet, a channel that it can be waited on for.
// Exactly one of the globals or channel will be returned.
// The final return value is true if this was the first request awaiting this key.
func (m *subincludeMap) Get(key string) (pyDict, <-chan struct{}, bool) {
	m.l.Lock()
	defer m.l.Unlock()
	if v, ok := m.m[key]; ok {
		return v.Globals, v.Wait, false
	}
	ch := make(chan struct{})
	m.m[key] = subincludeResult{Wait: ch}
	return nil, ch, true
}
