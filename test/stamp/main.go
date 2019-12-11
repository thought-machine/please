// Package main implements a simple binary that prints out the git revision
// at the time it was compiled.
package main

import (
	"fmt"

	"github.com/thought-machine/please/test/stamp/lib"
)

func main() {
	fmt.Println(lib.GitRevision)
	fmt.Println(lib.GitDescribe)
}
