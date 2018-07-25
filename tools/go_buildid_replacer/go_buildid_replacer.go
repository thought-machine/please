package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	args := os.Args
	for i, arg := range args {
		if arg == "-buildid" {
			args[i+1] = "BuiltWithPleaseBuild/BuiltWithPleaseBuild"
		}
	}
	err := syscall.Exec(args[1], args[1:], os.Environ())
	if err != nil {
		panic(fmt.Errorf("It has all gone wrong - %s", err))
	}
}
