// Package asm implements a simple test of Go assembly.
package asm

// add is the forward declaration of the assembly implementation.
func add(x, y int64) int64

// Add adds two numbers using assembly.
func Add(a, b int) int {
	return int(add(int64(a), int64(b)))
}
