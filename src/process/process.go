// Package process implements generic subprocess management functions.
package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
)

var log = logging.MustGetLogger("progress")

// An Executor handles starting, running and monitoring a set of subprocesses.
// It registers as a signal handler to attempt to terminate them all at process exit.
type Executor struct {
	sandboxCommand string
	processes      map[*exec.Cmd]struct{}
	mutex          sync.Mutex
}

// New returns a new Executor.
func New(sandboxCommand string) *Executor {
	o := &Executor{
		sandboxCommand: sandboxCommand,
		processes:      map[*exec.Cmd]struct{}{},
	}
	cli.AtExit(o.killAll) // Kill any subprocess if we are ourselves killed
	return o
}

// A Target is a minimal interface of what we need from a BuildTarget.
// It's here to avoid a hard dependency on the core package.
type Target interface {
	// String returns a string representation of this target.
	String() string
	// ShouldShowProgress returns true if the target should display progress.
	ShouldShowProgress() bool
	// SetProgress sets the current progress of the target.
	SetProgress(float32)
	// ProgressDescription returns a description of what the target is doing as it runs.
	ProgressDescription() string
}

// ExecWithTimeout runs an external command with a timeout.
// If the command times out the returned error will be a context.DeadlineExceeded error.
// If showOutput is true then output will be printed to stderr as well as returned.
// It returns the stdout only, combined stdout and stderr and any error that occurred.
func (e *Executor) ExecWithTimeout(target Target, dir string, env []string, timeout time.Duration, showOutput, attachStdin, attachStdout bool, argv []string) ([]byte, []byte, error) {
	// We deliberately don't attach this context to the command, so we have better
	// control over how the process gets terminated.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := e.ExecCommand(argv[0], argv[1:]...)
	defer e.removeProcess(cmd)
	cmd.Dir = dir
	cmd.Env = env

	var out bytes.Buffer
	var outerr safeBuffer
	var progress *float32
	if showOutput {
		cmd.Stdout = io.MultiWriter(os.Stderr, &out, &outerr)
		cmd.Stderr = io.MultiWriter(os.Stderr, &outerr)
	} else {
		cmd.Stdout = io.MultiWriter(&out, &outerr)
		cmd.Stderr = &outerr
	}
	if target != nil && target.ShouldShowProgress() {
		progress = new(float32)
		cmd.Stdout = newProgressWriter(target, progress, cmd.Stdout)
		cmd.Stderr = newProgressWriter(target, progress, cmd.Stderr)
	}
	if attachStdin {
		cmd.Stdin = os.Stdin
	}
	if attachStdout {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if target != nil {
		go logProgress(ctx, target, progress)
	}
	// Start the command, wait for the timeout & then kill it.
	// We deliberately don't use CommandContext because it will only send SIGKILL which
	// child processes can't handle themselves.
	err := cmd.Start()
	if err != nil {
		return nil, nil, err
	}
	ch := make(chan error)
	go runCommand(cmd, ch)
	select {
	case err = <-ch:
		// Do nothing.
	case <-time.After(timeout):
		e.KillProcess(cmd)
		err = fmt.Errorf("Timeout exceeded: %s", outerr.String())
	}
	return out.Bytes(), outerr.Bytes(), err
}

// runCommand runs a command and signals on the given channel when it's done.
func runCommand(cmd *exec.Cmd, ch chan error) {
	ch <- cmd.Wait()
}

// ExecWithTimeoutShell runs an external command within a Bash shell.
// Other arguments are as ExecWithTimeout.
// Note that the command is deliberately a single string.
func (e *Executor) ExecWithTimeoutShell(target Target, dir string, env []string, timeout time.Duration, showOutput bool, cmd string, sandbox bool) ([]byte, []byte, error) {
	return e.ExecWithTimeoutShellStdStreams(target, dir, env, timeout, showOutput, cmd, sandbox, false)
}

// ExecWithTimeoutShellStdStreams is as ExecWithTimeoutShell but optionally attaches stdin to the subprocess.
func (e *Executor) ExecWithTimeoutShellStdStreams(target Target, dir string, env []string, timeout time.Duration, showOutput bool, cmd string, sandbox, attachStdStreams bool) ([]byte, []byte, error) {
	c := append([]string{"bash", "--noprofile", "--norc", "-u", "-o", "pipefail", "-c"}, cmd)
	if sandbox {
		if e.sandboxCommand == "" {
			log.Fatalf("Sandbox tool not found on PATH")
		}
		c = append([]string{e.sandboxCommand}, c...)
	}
	return e.ExecWithTimeout(target, dir, env, timeout, showOutput, attachStdStreams, attachStdStreams, c)
}

// KillProcess kills a process, attempting to send it a SIGTERM first followed by a SIGKILL
// shortly after if it hasn't exited.
func (e *Executor) KillProcess(cmd *exec.Cmd) {
	success := killProcess(cmd, syscall.SIGTERM, 30*time.Millisecond)
	if !killProcess(cmd, syscall.SIGKILL, time.Second) && !success {
		log.Error("Failed to kill inferior process")
	}
	e.removeProcess(cmd)
}

func (e *Executor) removeProcess(cmd *exec.Cmd) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	delete(e.processes, cmd)
}

// killProcess implements the two-step killing of processes with a SIGTERM and a SIGKILL if
// that's unsuccessful. It returns true if the process exited within the timeout.
func killProcess(cmd *exec.Cmd, sig syscall.Signal, timeout time.Duration) bool {
	if cmd.Process == nil {
		log.Debug("Not terminating process, it seems to have not started yet")
		return false
	}
	// This is a bit of a fiddle. We want to wait for the process to exit but only for just so
	// long (we do not want to get hung up if it ignores our SIGTERM).
	log.Debug("Sending signal %s to -%d", sig, cmd.Process.Pid)
	syscall.Kill(-cmd.Process.Pid, sig) // Kill the group - we always set one in ExecCommand.
	ch := make(chan error, 1)
	go runCommand(cmd, ch)
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

// LogProgress logs a message once a minute until the given context has expired.
// Used to provide some notion of progress while waiting for external commands.
func (e *Executor) LogProgress(ctx context.Context, target Target) {
	logProgress(ctx, target, nil)
}

func logProgress(ctx context.Context, target Target, progress *float32) {
	name := target.String()
	msg := target.ProgressDescription()
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for i := 1; i < 1000000; i++ {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if i == 1 {
				log.Notice("%s still %s after 1 minute %s", name, msg, progressMessage(progress))
			} else {
				log.Notice("%s still %s after %d minutes %s", name, msg, i, progressMessage(progress))
			}
		}
	}
}

// safeBuffer is an io.Writer that ensures that only one thread writes to it at a time.
// This is important because we potentially have both stdout and stderr writing to the same
// buffer, and os.exec only guarantees goroutine-safety if both are the same writer, which in
// our case they're not (but are both ultimately causing writes to the same buffer)
type safeBuffer struct {
	sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(b []byte) (int, error) {
	sb.Lock()
	defer sb.Unlock()
	return sb.buf.Write(b)
}

func (sb *safeBuffer) Bytes() []byte {
	return sb.buf.Bytes()
}

func (sb *safeBuffer) String() string {
	return sb.buf.String()
}

// progressMessage displays a progress message, if it is being tracked.
func progressMessage(progress *float32) string {
	if progress != nil {
		return fmt.Sprintf("(%0.1f%% done)", *progress)
	}
	return ""
}

// killAll kills all subprocesses of this executor.
func (e *Executor) killAll() {
	e.mutex.Lock()
	processes := make([]*exec.Cmd, 0, len(e.processes))
	for proc := range e.processes {
		processes = append(processes, proc)
	}
	e.mutex.Unlock()

	if len(processes) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(processes))
		for _, proc := range processes {
			go func(proc *exec.Cmd) {
				e.KillProcess(proc)
				wg.Done()
			}(proc)
		}
		wg.Wait()
	}
}

// ExecCommand is a utility function that runs the given command with few options.
func ExecCommand(args ...string) ([]byte, error) {
	e := New("")
	cmd := e.ExecCommand(args[0], args[1:]...)
	defer e.removeProcess(cmd)
	return cmd.CombinedOutput()
}
