package main

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
)

func main() {
	fmt.Print(cmp.Equal(1, 1))
}
