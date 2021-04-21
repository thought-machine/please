package sandbox

import (
	"github.com/thought-machine/please/src/process"
	"os"
)

// Unshare will run the given program attaching this processes std in/out/err. It uses the process wrapper which will
// set up namespacing. This is probably not that useful except if you want to poke around in plz-out in an equivalent
// elevation.
//
// plz unshare bash # starts bash in an
func Unshare(args []string) error {
	e := process.New(true)
	cmd := e.ExecCommand(true, true, args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// TODO(jpoole): Read the docs. Attaching stdin and out doesn't seem to work with this.
	cmd.SysProcAttr.Setpgid = false

	return cmd.Run()
}