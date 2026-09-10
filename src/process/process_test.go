package process

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// argv returns a command that runs the given shell snippet, as an explicit argv rather than a
// shell string. Built through the shell rather than naming true, false and sleep directly:
// those are programs on the PATH on Unix and applets inside the shell on Windows, so only one
// of the two spellings works anywhere.
func argv(command string) []string {
	return New().BashCommand(command, false)
}

func TestExecWithTimeout(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 10*time.Second, false, false, false, false, NoSandbox, argv("exit 0"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutFailure(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 10*time.Second, false, false, false, false, NoSandbox, argv("exit 1"))
	assert.Error(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutDeadline(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 1*time.Nanosecond, false, false, false, false, NoSandbox, argv("sleep 10"))
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 0, len(out))
}

// TestKillsProcessTree covers the thing process groups on Unix and job objects on Windows both
// exist for: when a command times out, what it started has to die with it. Nothing else tests
// that on any platform, and the Windows implementation of it is entirely separate code.
func TestKillsProcessTree(t *testing.T) {
	// Forward slashes: this path is going into a shell command, where a backslash escapes.
	marker := filepath.ToSlash(filepath.Join(t.TempDir(), "marker"))
	// A grandchild that outlives the child it was started from, unless the whole tree is
	// killed. Deliberately a separate process rather than a subshell: busybox on Windows
	// implements a subshell as a thread, so it would die with its parent either way and prove
	// nothing about killing a tree.
	cmd := fmt.Sprintf("sh -c 'sleep 2; echo alive > %s' & sleep 30", marker)

	_, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 100*time.Millisecond, false, false, false, false, NoSandbox, argv(cmd))
	require.Error(t, err)

	// Comfortably past when the grandchild would have written, had it survived.
	time.Sleep(4 * time.Second)
	_, err = os.Stat(marker)
	assert.True(t, os.IsNotExist(err), "grandchild survived the timeout and wrote %s", marker)
}

func TestExecWithTimeoutOutput(t *testing.T) {
	targ := &target{}
	out, stderr, err := New().ExecWithTimeoutShell(targ, "", nil, 10*time.Second, false, false, NoSandbox, "echo hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestExecWithTimeoutStderr(t *testing.T) {
	targ := &target{}
	out, stderr, err := New().ExecWithTimeoutShell(targ, "", nil, 10*time.Second, false, false, NoSandbox, "echo hello 1>&2")
	assert.NoError(t, err)
	assert.Equal(t, "", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestBashCommandUsesConfiguredShell(t *testing.T) {
	e := NewSandboxingExecutor(false, NamespaceNever, "", "/bin/dash", []string{"--posix"})
	assert.Equal(t, []string{"/bin/dash", "--posix", "-e", "-u", "-o", "pipefail", "-c", "echo hello"},
		e.BashCommand("echo hello", true))
	assert.Equal(t, []string{"/bin/dash", "--posix", "-u", "-o", "pipefail", "-c", "echo hello"},
		e.BashCommand("echo hello", false))
}

func TestBashCommandDropsEmptyShellArgs(t *testing.T) {
	// A repeatable config key can't be cleared by assigning it empty; that yields a single
	// empty string, which must not reach the shell as an argument.
	e := NewSandboxingExecutor(false, NamespaceNever, "", "bash", []string{""})
	assert.Equal(t, []string{"bash", "-u", "-o", "pipefail", "-c", "echo hello"},
		e.BashCommand("echo hello", false))
}

func TestRemoteBashCommandIgnoresLocalShellArgs(t *testing.T) {
	// The remote worker runs a real bash whatever we're running on, so it keeps the full set
	// of flags regardless of how the local shell is configured.
	assert.Equal(t, []string{"bash", "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", "echo hello"},
		RemoteBashCommand("bash", "echo hello", false))
}
