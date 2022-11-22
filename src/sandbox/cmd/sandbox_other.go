//go:build !linux
// +build !linux

package main

import (
	"syscall"
)

func main() error {
	return syscall.Exec(os.Args[1], os.Args[2:]...)
}
