package exec

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Executor executes the command using the os/exec package
type Executor struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Run runs the command
func (e *Executor) Run(cmdStr string, args ...interface{}) error {
	cmdStr = fmt.Sprintf(cmdStr, args...)
	fmt.Fprintf(os.Stderr, "please_go install -> %v\n", cmdStr)

	cmd := exec.Command("bash", "-e", "-c", cmdStr)
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr

	return cmd.Run()
}

func (e *Executor) CombinedOutput(bin string, args ...string) ([]byte, error) {
	fmt.Fprintln(os.Stderr, "please_go install ->", bin, strings.Join(args, " "))

	cmd := exec.Command(bin, args...)
	return cmd.CombinedOutput()
}
