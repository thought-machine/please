package process

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecWithTimeout(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 10*time.Second, false, false, false, false, []string{"true"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutFailure(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 10*time.Second, false, false, false, false, []string{"false"})
	assert.Error(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutDeadline(t *testing.T) {
	out, _, err := New().ExecWithTimeout(context.Background(), nil, "", nil, 1*time.Nanosecond, false, false, false, false, []string{"sleep", "10"})
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutOutput(t *testing.T) {
	targ := &target{}
	out, stderr, err := New().ExecWithTimeoutShell(targ, "", nil, 10*time.Second, false, false, "echo hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestExecWithTimeoutStderr(t *testing.T) {
	targ := &target{}
	out, stderr, err := New().ExecWithTimeoutShell(targ, "", nil, 10*time.Second, false, false, "echo hello 1>&2")
	assert.NoError(t, err)
	assert.Equal(t, "", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestKillSubprocesses(t *testing.T) {
	e := New()
	cmd := e.ExecCommand(false, "sleep", "infinity")
	e.registerProcess(cmd)
	assert.Equal(t, 1, len(e.processes))
	err := cmd.Start()
	assert.NoError(t, err)
	e.killAll()
	err = cmd.Wait()
	assert.Error(t, err)
	assert.Equal(t, 0, len(e.processes))
}
