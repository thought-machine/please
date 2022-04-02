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

func TestKillSubprocesses(t *testing.T) {
	e := New()
	ch := make(chan error)
	go func() {
		_, _, err := e.ExecWithTimeout(context.Background(), nil, "", nil, time.Hour, false, false, false, false, NoSandbox, []string{"sleep", "infinity"})
		ch <- err
	}()
	// Check that it doesn't error immediately
	select {
	case err := <-ch:
		t.Fatalf("Unexpected error from executor: %s", err)
	case <-time.After(10 * time.Millisecond):
	}
	// Now kill it
	e.killAll()
	err := <-ch
	assert.Error(t, err)
}
