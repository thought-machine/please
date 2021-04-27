package sandbox

import (
	"fmt"
	"github.com/thought-machine/please/src/process"
	"os"
	"strings"
	"syscall"
)

// Unshare will run the given program attaching this processes std in/out/err. It uses the process wrapper which will
// set up namespacing. This is probably not that useful except if you want to poke around in plz-out in an equivalent
// elevation.
//
// For example, `plz unshare bash` will open up a bash shell as a fake root user in a namespaced environment.
func Unshare(args []string) error {
	e := process.NewSandboxingExecutor(true, false, "")
	cmd := e.ExecCommand(false, args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Add in more sandboxing so `plz unshare plz sandbox` works.
	cmd.SysProcAttr.Cloneflags = cmd.SysProcAttr.Cloneflags | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS

	// TODO(jpoole): Read the docs. Attaching stdin and out doesn't seem to work with this.
	cmd.SysProcAttr.Setpgid = false

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run '%s': %w", strings.Join(args, " "), err)
	}
	return nil
}
