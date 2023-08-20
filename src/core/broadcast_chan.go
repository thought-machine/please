package core

import ()

// A BroadcastChan is like a channel but supports setting a value once, which is then visible to all callers.
type BroadcastChan[T any] struct {
	ch chan struct{}
	t  T
}

// NewBroadcastChan creates a new BroadcastChan
func NewBroadcastChan[T any]() BroadcastChan[T] {
	return BroadcastChan[T]{ch: make(chan struct{})}
}

// Wait waits for someone to call Complete then returns its value
// Any number of concurrent callers may call Wait() simultaneously.
func (ch *BroadcastChan[T]) Wait() T {
	<-ch.ch
	return ch.t
}

// Complete marks the chan as complete. Any callers waiting on `Wait()` will receive the value passed in here.
// It is not threadsafe to call this multiple times concurrently.
func (ch *BroadcastChan[T]) Complete(t T) {
	ch.t = t
	close(ch.ch)
}
