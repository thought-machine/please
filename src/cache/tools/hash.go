package tools

import "math"

// HashPoint returns a point in our hash space for the ith of n nodes.
// Note that it is quite important that both client and server agree
// about the implementation of this function.
func HashPoint(i, n int) uint32 {
	return i * (math.MaxUint32 / n)
}
