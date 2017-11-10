// Package asm implements a simple binary using Go assembly.
package main

import (
	"fmt"
	"os"
	"strconv"
)

// add is the forward declaration of the assembly implementation.
func add(x, y int64) int64

func main() {
	x, _ := strconv.ParseInt(os.Args[1], 10, 64)
	y, _ := strconv.ParseInt(os.Args[2], 10, 64)
	fmt.Printf("%d\n", add(x, y))
}
