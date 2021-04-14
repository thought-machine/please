package exec

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Executor are able to "execute" the command.
type Executor interface {
	Exec(cmd string, args ...interface{}) error
}

// OsExecutor executes the command using the os/exec package
type OsExecutor struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Exec runs the command
func (e *OsExecutor) Exec(cmdStr string, args ...interface{}) error {
	cmdStr = fmt.Sprintf(cmdStr, args...)
	fmt.Fprintf(os.Stderr, "please_go install -> %v\n", cmdStr)

	cmd := exec.Command("bash", "-e", "-c", cmdStr)
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr

	return cmd.Run()
}
