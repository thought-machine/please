package exec

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

// Executor are able to "execute" the command.
type Executor interface {
	Exec(cmd string, args ...interface{})
}

// OsExecutor executes the command using the os/exec package
type OsExecutor struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Exec runs the command
func (e *OsExecutor) Exec(cmdStr string, args ...interface{}) {
	cmdStr = fmt.Sprintf(cmdStr, args...)
	fmt.Fprintf(os.Stderr, "please_go_install -> %v\n", cmdStr)

	cmd := exec.Command("bash", "-e", "-c", cmdStr)
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute cmd: %v", err)
	}
}
