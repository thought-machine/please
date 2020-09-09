// Package asm implements a simple test of Go assembly.
package asm

import "github.com/thought-machine/please/test/go_rules/asm/golib"

// add is the forward declaration of the assembly implementation.
func add(x, y int64) int64

// Add adds two numbers using assembly.
func Add() int {
	return int(add(golib.LHS, golib.RHS))
}
