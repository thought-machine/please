package sandbox

import (
	"github.com/thought-machine/please/src/process"
	"os"
)

// Unshare will run the given program attaching this processes std in/out/err. It uses the process wrapper which will
// set up namespacing. This is probably not that useful except if you want to poke around in plz-out in an equivalent
// elevation.
//
// For example, `plz unshare bash` will open up a bash shell as a fake root user in a namespaced environment.
func Unshare(args []string) error {
	e := process.NewSandboxingExecutor(true, false, "")
	cmd := e.ExecCommand(true, args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// TODO(jpoole): Read the docs. Attaching stdin and out doesn't seem to work with this.
	cmd.SysProcAttr.Setpgid = false

	return cmd.Run()
}
