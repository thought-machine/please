package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func main() {
	args := os.Args
	for i, arg := range args {
		if arg == "-buildid" {
			args[i+1] = "BuiltWithPleaseBuild/BuiltWithPleaseBuild"
		}
	}
	cmd := exec.Command(args[1], args[2:]...)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Errorf("It has all gone wrong - %s", err)
	}
	io.Copy(os.Stdout, bytes.NewReader(stdout))
}
