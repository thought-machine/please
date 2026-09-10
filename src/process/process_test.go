package process

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecWithTimeout(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 10*time.Second, false, false, false, false, NoSandbox, []string{"true"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutFailure(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 10*time.Second, false, false, false, false, NoSandbox, []string{"false"})
	assert.Error(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutDeadline(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 1*time.Nanosecond, false, false, false, false, NoSandbox, []string{"sleep", "10"})
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 0, len(out))
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
