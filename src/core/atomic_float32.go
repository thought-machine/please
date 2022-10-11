package core

import (
	"math"
	"sync/atomic"
)

// Inspired by the go.uber.org/atomic Float32 type

type atomicFloat32 struct {
	v uint32
}

func (f *atomicFloat32) Load() float32 {
	return math.Float32frombits(atomic.LoadUint32(&f.v))
}

func (f *atomicFloat32) Store(val float32) {
	atomic.StoreUint32(&f.v, math.Float32bits(val))
}
