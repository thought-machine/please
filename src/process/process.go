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

	"github.com/thought-machine/please/src/cli/logging"

	"github.com/thought-machine/please/src/cli"
)

var log = logging.Log

type NamespacingPolicy string

const (
	NamespaceAlways  NamespacingPolicy = "always"
	NamespaceNever   NamespacingPolicy = "never"
	NamespaceSandbox NamespacingPolicy = "sandbox"
)

// An Executor handles starting, running and monitoring a set of subprocesses.
// It registers as a signal handler to attempt to terminate them all at process exit.
type Executor struct {
	// Whether we should set up linux namespaces at all
	namespace NamespacingPolicy
	// The tool that will do the network/mount sandboxing
	sandboxTool      string
	usePleaseSandbox bool
	processes        map[*exec.Cmd]<-chan error
	mutex            sync.Mutex
}

func NewSandboxingExecutor(usePleaseSandbox bool, namespace NamespacingPolicy, sandboxTool string) *Executor {
	o := &Executor{
		namespace:        namespace,
		usePleaseSandbox: usePleaseSandbox,
		sandboxTool:      sandboxTool,
		processes:        map[*exec.Cmd]<-chan error{},
	}
	cli.AtExit(o.killAll) // Kill any subprocess if we are ourselves killed
	return o
}

// New returns a new Executor.
func New() *Executor {
	return NewSandboxingExecutor(false, NamespaceNever, "")
}

// SandboxConfig contains what namespaces should be sandboxed
type SandboxConfig struct {
	Network, Mount, Fakeroot bool
}

// NoSandbox represents a no-sandbox value
var NoSandbox = SandboxConfig{}

// NewSandboxConfig creates a new SandboxConfig
func NewSandboxConfig(network, mount bool) SandboxConfig {
	return SandboxConfig{Network: network, Mount: mount}
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
	// ShouldExitOnError returns true if the executed process should exit if an error occurs.
	ShouldExitOnError() bool
}

// ExecWithTimeout runs an external command with a timeout.
// If the command times out the returned error will be a context.DeadlineExceeded error.
// If showOutput is true then output will be printed to stderr as well as returned.
// It returns the stdout only, combined stdout and stderr and any error that occurred.
func (e *Executor) ExecWithTimeout(ctx context.Context, target Target, dir string, env []string, timeout time.Duration, showOutput, attachStdin, attachStdout, foreground bool, sandbox SandboxConfig, argv []string) ([]byte, []byte, error) {
	// We deliberately don't attach this context to the command, so we have better
	// control over how the process gets terminated.
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := e.ExecCommand(sandbox, foreground, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, env...)

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
	e.registerProcess(cmd, ch)
	defer e.removeProcess(cmd)
	go runCommand(cmd, ch)
	select {
	case err = <-ch:
		// Do nothing.
	case <-ctx.Done():
		err = ctx.Err()
		e.KillProcess(cmd)
	}
	return out.Bytes(), outerr.Bytes(), err
}

// runCommand runs a command and signals on the given channel when it's done.
func runCommand(cmd *exec.Cmd, ch chan<- error) {
	ch <- cmd.Wait()
}

// ExecWithTimeoutShell runs an external command within a Bash shell.
// Other arguments are as ExecWithTimeout.
// Note that the command is deliberately a single string.
func (e *Executor) ExecWithTimeoutShell(target Target, dir string, env []string, timeout time.Duration, showOutput, foreground bool, sandbox SandboxConfig, cmd string) ([]byte, []byte, error) {
	return e.ExecWithTimeoutShellStdStreams(target, dir, env, timeout, showOutput, foreground, sandbox, cmd, false)
}

// ExecWithTimeoutShellStdStreams is as ExecWithTimeoutShell but optionally attaches stdin to the subprocess.
func (e *Executor) ExecWithTimeoutShellStdStreams(target Target, dir string, env []string, timeout time.Duration, showOutput, foreground bool, sandbox SandboxConfig, cmd string, attachStdStreams bool) ([]byte, []byte, error) {
	c := BashCommand("bash", cmd, target.ShouldExitOnError())
	return e.ExecWithTimeout(context.Background(), target, dir, env, timeout, showOutput, attachStdStreams, attachStdStreams, foreground, sandbox, c)
}

// KillProcess kills a process, attempting to send it a SIGTERM first followed by a SIGKILL
// shortly after if it hasn't exited.
func (e *Executor) KillProcess(cmd *exec.Cmd) {
	e.killProcess(cmd, e.processChan(cmd))
}

func (e *Executor) killProcess(cmd *exec.Cmd, ch <-chan error) {
	success := sendSignal(cmd, ch, syscall.SIGTERM, 30*time.Millisecond)
	if !sendSignal(cmd, ch, syscall.SIGKILL, time.Second) && !success {
		log.Error("Failed to kill inferior process")
	}
	e.removeProcess(cmd)
}

func (e *Executor) removeProcess(cmd *exec.Cmd) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	delete(e.processes, cmd)
}

// registerProcess stores the given process in this executor's map.
func (e *Executor) registerProcess(cmd *exec.Cmd, ch <-chan error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.processes[cmd] = ch
}

// processChan returns the error channel for a process.
func (e *Executor) processChan(cmd *exec.Cmd) <-chan error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.processes[cmd]
}

// sendSignal sends a single signal to the process in an attempt to stop it.
// It returns true if the process exited within the timeout.
func sendSignal(cmd *exec.Cmd, ch <-chan error, sig syscall.Signal, timeout time.Duration) bool {
	if cmd.Process == nil {
		log.Debug("Not terminating process, it seems to have not started yet")
		return false
	}
	// This is a bit of a fiddle. We want to wait for the process to exit but only for just so
	// long (we do not want to get hung up if it ignores our SIGTERM).
	log.Debug("Sending signal %s to -%d", sig, cmd.Process.Pid)
	syscall.Kill(-cmd.Process.Pid, sig) // Kill the group - we always set one in ExecCommand.

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
	var wg sync.WaitGroup
	wg.Add(len(e.processes))
	defer wg.Wait()
	defer e.mutex.Unlock()
	for proc, ch := range e.processes {
		go func(proc *exec.Cmd, ch <-chan error) {
			e.killProcess(proc, ch)
			wg.Done()
		}(proc, ch)
	}
}

// ExecCommand is a utility function that runs the given command with few options.
func ExecCommand(args ...string) ([]byte, error) {
	e := New()
	cmd := e.ExecCommand(NoSandbox, false, args[0], args[1:]...)
	defer e.removeProcess(cmd)
	return cmd.CombinedOutput()
}

// BashCommand returns the command that we'd use to execute a subprocess in a shell with.
func BashCommand(binary, command string, exitOnError bool) []string {
	if exitOnError {
		return []string{binary, "--noprofile", "--norc", "-e", "-u", "-o", "pipefail", "-c", command}
	}
	return []string{binary, "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", command}
}
