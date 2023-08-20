package core

import (
	"sync"
	"sync/atomic"
)

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
// This function may only be called once.
func (ch *BroadcastChan[T]) Complete(t T) {
	ch.t = t
	close(ch.ch)
}

// An initialErrgroup is like errgroup.Group but immediately returns the first error encountered during processing
type initialErrgroup struct {
	ch    BroadcastChan[error]
	count atomic.Int64
	once  sync.Once
}

func (ie *initialErrgroup) Go(f func() error) {
	if ie.ch.ch == nil {
		ie.ch = NewBroadcastChan[error]()
	}
	ie.count.Add(1)
	go func() {
		defer func() {
			if ie.count.Add(-1) == 0 {
				ie.complete(nil)
			}
		}()
		if err := f(); err != nil {
			ie.complete(err)
		}
	}()
}

func (ie *initialErrgroup) Wait() error {
	return ie.ch.Wait()
}

func (ie *initialErrgroup) complete(err error) {
	ie.once.Do(func() {
		ie.ch.Complete(err)
	})
}
