package tools

import (
	"encoding/binary"
	"math"
)

// HashPoint returns a point in our hash space for the ith of n nodes.
// Note that it is quite important that both client and server agree
// about the implementation of this function.
func HashPoint(i, n int) uint32 {
	return uint32(i * math.MaxUint32 / n)
}

// Hash returns the point in our hash space for a given artifact hash.
func Hash(h []byte) uint32 {
	return binary.LittleEndian.Uint32(h)
}

// AlternateHash returns the alternate point in our hash space for a given artifact hash,
// i.e. on the second node we'd replicate it to.
func AlternateHash(h []byte) uint32 {
	const halfway = 1 << 31
	point := Hash(h)
	if point > halfway {
		return point - halfway
	}
	return point + halfway
}
