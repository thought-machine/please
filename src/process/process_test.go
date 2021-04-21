package process

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecWithTimeout(t *testing.T) {
	out, _, err := New(false).ExecWithTimeout(nil, "", nil, 10*time.Second, false, false, false, []string{"true"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutFailure(t *testing.T) {
	out, _, err := New(false).ExecWithTimeout(nil, "", nil, 10*time.Second, false, false, false, []string{"false"})
	assert.Error(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutDeadline(t *testing.T) {
	out, _, err := New(false).ExecWithTimeout(nil, "", nil, 1*time.Nanosecond, false, false, false, []string{"sleep", "10"})
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "Timeout exceeded"))
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutOutput(t *testing.T) {
	targ := &target{}
	out, stderr, err := New(false).ExecWithTimeoutShell(targ, "", nil, 10*time.Second, false, "echo hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestExecWithTimeoutStderr(t *testing.T) {
	targ := &target{}
	out, stderr, err := New(false).ExecWithTimeoutShell(targ, "", nil, 10*time.Second, false, "echo hello 1>&2")
	assert.NoError(t, err)
	assert.Equal(t, "", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestKillSubprocesses(t *testing.T) {
	e := New(false)
	cmd := e.ExecCommand(false, false, "sleep", "infinity")
	assert.Equal(t, 1, len(e.processes))
	err := cmd.Start()
	assert.NoError(t, err)
	e.killAll()
	err = cmd.Wait()
	assert.Error(t, err)
	assert.Equal(t, 0, len(e.processes))
}
