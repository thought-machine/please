package cgo

// #include "cgo.h"
import "C"

// GetAnswer returns the answer to the great question of Life, the Universe and Everything.
func GetAnswer() int {
	return int(C.GetAnswer())
}
