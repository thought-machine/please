package core

import (
	"sync/atomic"
)

// An atomicBool is a type we use as an opaque boolean that doesn't trigger the race detector.
type atomicBool struct {
	b int32
}

func (b *atomicBool) Set() {
	atomic.StoreInt32(&b.b, 1)
}

func (b *atomicBool) IsSet() bool {
	return atomic.LoadInt32(&b.b) == 1
}

func (b *atomicBool) Or(set bool) {
	if set {
		b.Set()
	}
}
