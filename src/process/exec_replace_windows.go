package process

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
)

// ExecReplace replaces the currently running process with the given command.
// It does not return unless the exec itself failed.
//
// Windows has no way to replace a process image, so we run the command as a child, wait for
// it, and exit with its status. Two consequences callers need to be aware of:
//
//   - We stay alive as the child's parent. Any resource we hold is still held, so release
//     anything the child will contend for - notably the repo lock - before calling this. On
//     Unix that happens implicitly, because Go opens files O_CLOEXEC and the exec releases
//     the lock for us.
//   - Nothing deferred in the caller runs, matching execve.
func ExecReplace(path string, argv, env []string) error {
	cmd := exec.Command(path)
	cmd.Args = argv
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// The child shares our console, so a Ctrl-C reaches it directly. Ignore it here so we
	// don't exit first and leave it writing to a console nobody is reading.
	signal.Ignore(os.Interrupt)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil // unreachable
}
