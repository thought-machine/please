//go:build !linux
// +build !linux

package sandbox

import (
	"os"
	"os/exec"
)

func Sandbox(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
