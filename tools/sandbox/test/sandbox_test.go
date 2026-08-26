package sandbox_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func runSandbox(t *testing.T, sandbox string, args []string, env []string) string {
	t.Helper()
	cmd := exec.Command(sandbox, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "sandbox command failed: %q, output: %q", err, string(out))
	return strings.TrimSpace(string(out))
}

func pleaseSandbox(t *testing.T, args []string, env []string) string {
	t.Helper()
	return runSandbox(t, os.Getenv("DATA_PLEASE_SANDBOX"), args, env)
}

// TestSandboxEnvMatrix enumerates every combination of the environment variables that select
// which namespaces the sandbox creates, asserting each dimension independently: whether the
// network and mount namespaces are new, and whether a fresh /proc hides outer processes.
func TestSandboxEnvMatrix(t *testing.T) {
	outerNet, err := os.Readlink("/proc/self/ns/net")
	require.NoError(t, err)
	outerMnt, err := os.Readlink("/proc/self/ns/mnt")
	require.NoError(t, err)

	// The test's own pid exists in the outer /proc but not in a freshly remounted one.
	probe := []string{"sh", "-c", fmt.Sprintf(
		"readlink /proc/self/ns/net; readlink /proc/self/ns/mnt; test -d /proc/%d && echo visible || echo hidden",
		os.Getpid())}

	for _, shareNetwork := range []bool{false, true} {
		for _, shareMount := range []bool{false, true} {
			for _, mountProc := range []bool{true, false} {
				var opts []string
				if shareNetwork {
					opts = append(opts, "SHARE_NETWORK=1")
				}
				if shareMount {
					opts = append(opts, "SHARE_MOUNT=1")
				}
				if !mountProc {
					opts = append(opts, "MOUNT_PROC=0")
				}
				name := "default"
				if len(opts) > 0 {
					name = strings.Join(opts, ",")
				}

				t.Run(name, func(t *testing.T) {
					lines := strings.Split(pleaseSandbox(t, probe, append([]string{"TMP_DIR=/tmp"}, opts...)), "\n")
					require.Len(t, lines, 3)
					if shareNetwork {
						require.Equal(t, outerNet, lines[0], "network namespace should be shared")
					} else {
						require.NotEqual(t, outerNet, lines[0], "network namespace should be new")
					}
					if shareMount {
						require.Equal(t, outerMnt, lines[1], "mount namespace should be shared")
					} else {
						require.NotEqual(t, outerMnt, lines[1], "mount namespace should be new")
					}
					// /proc is only remounted when a new mount namespace exists and
					// MOUNT_PROC is not disabled.
					if !shareMount && mountProc {
						require.Equal(t, "hidden", lines[2], "fresh /proc should hide outer processes")
					} else {
						require.Equal(t, "visible", lines[2], "outer /proc should remain visible")
					}
				})
			}
		}
	}
}

func TestSandboxCommon(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Connection accepted"))
	})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	if _, err := os.Stat("/dev/null"); err != nil {
		t.Fatalf("Test precondition failed: failed to stat /dev/null: %v", err)
	}

	if _, err := os.Stat("/tmp"); err != nil {
		t.Fatalf("Test precondition failed: failed to stat /tmp: %v", err)
	}

	tests := []struct {
		name string
		env  []string
		args []string
		want string
	}{
		{
			name: "must share network when SHARE_NETWORK=1",
			env:  []string{"SHARE_NETWORK=1", "TMP_DIR=/tmp"},
			args: []string{"sh", "-c", "curl -sS " + ts.URL},
			want: "Connection accepted",
		},
		{
			name: "must isolate network when SHARE_NETWORK=0",
			env:  []string{"SHARE_NETWORK=0", "TMP_DIR=/tmp"},
			args: []string{"sh", "-c", fmt.Sprintf(`curl -s %s; echo "curl exit: $?"`, ts.URL)},
			want: "curl exit: 7", // exit code 7 represents "Failed to connect to host."
		},
		{
			name: "must map parent user to its own UID by default",
			env:  []string{"TMP_DIR=/tmp"},
			args: []string{"sh", "-c", "echo UID=$(id -u)"},
			want: fmt.Sprintf("UID=%d", os.Getuid()),
		},
		{
			name: "SANDBOX_DIRS must hide the contents of a specified directory",
			env:  []string{"TMP_DIR=/tmp", "SANDBOX_DIRS=/dev"},
			args: []string{"ls", "/dev"},
			want: "",
		},
		{
			name: "SANDBOX_FILE_MOUNTS must mount the specified file",
			env:  []string{"TMP_DIR=/tmp", "SANDBOX_FILE_MOUNTS=/proc/sys/kernel/ostype:/dev/null"},
			args: []string{"cat", "/dev/null"},
			want: "Linux",
		},
		{
			name: "SANDBOX_UID_MAP maps current uid/gid to 0",
			env: []string{
				"TMP_DIR=/tmp",
				fmt.Sprintf("SANDBOX_UID_MAP=0 %d 1", os.Getuid()),
				fmt.Sprintf("SANDBOX_GID_MAP=0 %d 1", os.Getgid()),
			},
			args: []string{"sh", "-c", "echo $(id -u)/$(id -g)"},
			want: "0/0",
		},
		{
			// Outside ids below 65536 so they exist even in a constrained container uid space (e.g. the default rootless mapping).
			name: "SANDBOX_UID_MAP uid and gid ranges",
			env:  []string{"TMP_DIR=/tmp", "SANDBOX_UID_MAP=0 20000 10   200 23000 40", "SANDBOX_GID_MAP=50 26000 700"},
			args: []string{"sh", "-c", "cat /proc/self/uid_map /proc/self/gid_map | awk '{$1=$1}1'"},
			want: "0 20000 10\n200 23000 40\n50 26000 700",
		},
		{
			name: "SANDBOX_UID_MAP uid and gid ranges allow us to use chown",
			env: []string{
				"TMP_DIR=/var",
				fmt.Sprintf("SANDBOX_UID_MAP=0 %d 1  1 20000 40000", os.Getuid()),
				fmt.Sprintf("SANDBOX_GID_MAP=0 %d 1  1 20000 40000", os.Getgid()),
			},
			args: []string{"sh", "-c", "touch /tmp/f && chown 200:50 /tmp/f && stat -c '%u %g' /tmp/f"},
			want: "200 50",
		},
		{
			name: "Sandbox denies setgroups by default",
			env:  []string{"TMP_DIR=/tmp"},
			args: []string{"cat", "/proc/self/setgroups"},
			want: "deny",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := pleaseSandbox(t, tt.args, tt.env)

			if output != tt.want {
				t.Errorf("Expected %q, but got %q", tt.want, output)
			}
		})
	}
}

func TestSandboxNoTmpDir(t *testing.T) {
	marker := fmt.Sprintf("/tmp/sandbox-test-marker-%d", os.Getpid())
	require.NoError(t, os.WriteFile(marker, []byte("outer"), 0o644))
	defer os.Remove(marker)

	probe := []string{"sh", "-c", fmt.Sprintf(
		`echo "TMPDIR=$TMPDIR"; test -e %s && echo marker-visible || echo marker-hidden; test -e /tmp/plz_sandbox && echo bind-mounted || echo no-bind-mount`,
		marker)}
	out := pleaseSandbox(t, probe, []string{})

	require.Contains(t, out, "TMPDIR=/tmp\n", "TMPDIR should point at the tmpfs mounted over /tmp")
	require.Contains(t, out, "marker-hidden", "a file in the outer /tmp should be hidden by the tmpfs mounted over it")
	require.Contains(t, out, "no-bind-mount", "nothing should be bind mounted at /tmp/plz_sandbox when TMP_DIR is unset")
}

func TestSandboxLocalIP(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		args []string
		want string
	}{
		{
			name: "adds 10.1.1.1 to the loopback interface by default",
			env:  []string{"TMP_DIR=/tmp"},
			args: []string{"sh", "-c", `grep -o '10\.1\.1\.1' /proc/net/fib_trie | head -n1`},
			want: "10.1.1.1",
		},
		{
			name: "SANDBOX_LOCAL_IP overrides the default address",
			env:  []string{"TMP_DIR=/tmp", "SANDBOX_LOCAL_IP=10.2.3.4"},
			args: []string{"sh", "-c", `grep -o '10\.2\.3\.4' /proc/net/fib_trie | head -n1; grep -c '10\.1\.1\.1' /proc/net/fib_trie; true`},
			want: "10.2.3.4\n0",
		},
		{
			name: "SANDBOX_LOCAL_IP= disables the extra address",
			env:  []string{"TMP_DIR=/tmp", "SANDBOX_LOCAL_IP="},
			args: []string{"sh", "-c", `grep -c '10\.' /proc/net/fib_trie; true`},
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := pleaseSandbox(t, tt.args, tt.env)

			if output != tt.want {
				t.Errorf("Expected %q, but got %q", tt.want, output)
			}
		})
	}
}

// TestSandboxNoOrphanOnParentCrash checks that the sandboxed command does not outlive the
// sandbox process itself. Unlike TestSandboxHangOnParentCrash, which tests the window
// between clone() and the sync pipe, this kills the sandbox after the command has exec'd,
// executing the PR_SET_PDEATHSIG path.
func TestSandboxNoOrphanOnParentCrash(t *testing.T) {
	sandbox := os.Getenv("DATA_PLEASE_SANDBOX")
	if sandbox == "" {
		t.Skip("DATA_PLEASE_SANDBOX not set")
	}

	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	// The sleep only needs to outlive the 2s assertion window below.
	cmd := exec.Command(sandbox, "sh", "-c", "echo ready && exec sleep 10")
	cmd.Env = []string{"TMP_DIR=/tmp"}
	cmd.Stdout = w
	require.NoError(t, cmd.Start())
	// Close our copy of the write end so the read below blocks only on the sandboxed command.
	w.Close()

	// Wait for the sandboxed command to be running, i.e. definitely past execvp.
	buf := make([]byte, 6)
	_, err = io.ReadFull(r, buf)
	require.NoError(t, err)
	require.Equal(t, "ready\n", string(buf))

	// Simulate a crash of the sandbox process.
	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()

	// The sandboxed process must die with it, closing its end of the pipe.
	errCh := make(chan error, 1)
	go func() {
		_, err := r.Read(make([]byte, 1))
		errCh <- err
	}()
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, io.EOF, "expected EOF once the sandboxed process died")
	case <-time.After(2 * time.Second):
		t.Fatal("sandboxed process survived the death of the sandbox process (orphan leak)")
	}
}

func TestSandboxHangOnParentCrash(t *testing.T) {
	sandbox := os.Getenv("DATA_PLEASE_SANDBOX")
	if sandbox == "" {
		t.Skip("DATA_PLEASE_SANDBOX not set")
	}

	// We run this in a loop with different delays to reliably hit the race condition window where
	// the parent dies after clone() but before writing to the sync pipe.
	for i := range 20 {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			r, w, err := os.Pipe()
			require.NoError(t, err)
			defer r.Close()

			cmd := exec.Command(sandbox, "echo", "success")
			cmd.Stdout = w

			err = cmd.Start()
			require.NoError(t, err)

			// Close our copy of the write-end immediately so that the read below can only block on either the sandbox itself or the sandboxed command.
			w.Close()

			// Sleep just long enough for the parent to call clone(),
			// but (hopefully) before it finishes uid mapping.
			time.Sleep(time.Duration(i) * 100 * time.Microsecond)

			_ = cmd.Process.Kill()
			_ = cmd.Wait()

			errCh := make(chan error, 1)
			go func() {
				buf := make([]byte, 128)
				_, err := r.Read(buf)
				errCh <- err
			}()

			select {
			case err := <-errCh:
				// Success: The read unblocked. The child either finished fast
				// or safely aborted when it noticed the parent died.
				if err != io.EOF && err != nil {
					t.Logf("Expected EOF, got: %v", err)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Child process hung indefinitely holding stdout after parent was killed! (Race condition triggered)")
			}
		})
	}
}
