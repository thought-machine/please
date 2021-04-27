package main

import (
	"os"
	"syscall"

	"github.com/thought-machine/please/src/process"
)

func main() {
	e := process.NewSandboxingExecutor(true, false, "")
	cmd := e.ExecCommand(false, "ls")

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Add in more sandboxing so `plz unshare plz sandbox` works.
	cmd.SysProcAttr.Cloneflags = cmd.SysProcAttr.Cloneflags | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS

	// TODO(jpoole): Read the docs. Attaching stdin and out doesn't seem to work with this.
	cmd.SysProcAttr.Setpgid = false

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
