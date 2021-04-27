package main

import (
	"github.com/thought-machine/please/src/sandbox"
	"os"
)

func main() {
	if err := sandbox.Sandbox(os.Args[1:]); err != nil {
		panic(err)
	}
}
