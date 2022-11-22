//go:build !linux
// +build !linux

package main

import (
	"os"
	"syscall"
)

func main() {
	if err := syscall.Exec(os.Args[1], os.Args[2:], os.Environ()); err != nil {
		panic(err)
	}
}
