// +build linux

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/thought-machine/please/src/core"
)

// mdLazytime is the bit for lazily flushing disk writes.
// TODO(jpoole): find out if there's a reason this isn't in syscall
const mdLazytime = 1 << 25

const sandboxDirsVar = "SANDBOX_DIRS"

var sandboxMountDir = core.SandboxDir

func Sandbox(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("incorrect number of args to call plz sandbox")
	}
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	unshareMount := os.Getenv("SHARE_MOUNT") != "1"
	unshareNetwork := os.Getenv("SHARE_NETWORK") != "1"

	if unshareMount {
		tmpDirEnv := os.Getenv("TMP_DIR")
		if tmpDirEnv == "" {
			return fmt.Errorf("$TMP_DIR is not set but required. It must contain the directory path to be sandboxed")
		}

		if err := sandboxDir(tmpDirEnv); err != nil {
			return err
		}

		if err := mountSandboxDirs(); err != nil {
			return fmt.Errorf("Failed to mount over sandboxed dirs: %w", err)
		}

		if err := rewriteEnvVars(tmpDirEnv, sandboxMountDir); err != nil {
			return err
		}

		cmd.Dir = sandboxMountDir

		if err := mountProc(); err != nil {
			return err
		}
	}

	if unshareNetwork {
		if err := loUp(); err != nil {
			return fmt.Errorf("Failed to bring loopback interface up: %s", err)
		}
	}

	return cmd.Run()
}

func rewriteEnvVars(from, to string) error {
	for _, envVar := range os.Environ() {
		if strings.Contains(envVar, from) {
			parts := strings.Split(envVar, "=")
			key := parts[0]
			value := strings.TrimPrefix(envVar, key+"=")
			if err := os.Setenv(key, strings.ReplaceAll(value, from, to)); err != nil {
				return err
			}
		}
	}
	return nil
}

func sandboxDir(dir string) error {
	if strings.HasPrefix(dir, "/tmp") {
		return fmt.Errorf("Not mounting /tmp as %s is a subdir", dir)
	}

	// Remounting / as private is necessary so that the tmpfs mount isn't visible to anyone else.
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("Failed to mount root: %w", err)
	}

	flags := mdLazytime | syscall.MS_NOATIME | syscall.MS_NODEV | syscall.MS_NOSUID
	if err := syscall.Mount("", "/tmp", "tmpfs", uintptr(flags), ""); err != nil {
		return fmt.Errorf("Failed to mount /tmp: %w", err)
	}

	if err := os.Setenv("TMPDIR", "/tmp"); err != nil {
		return fmt.Errorf("Failed to set $TMPDIR: %w", err)
	}

	if err := os.Mkdir(sandboxMountDir, os.ModeDir|0775); err != nil {
		return fmt.Errorf("Failed to make %s: %w", sandboxMountDir, err)
	}

	if err := syscall.Mount(dir, sandboxMountDir, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to bind %s to %s : %w", dir, sandboxMountDir, err)
	}

	if err := syscall.Mount("", "/", "", syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to remount root as readonly: %w", err)
	}

	return nil
}

func mountProc() error {
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("Failed to mount /proc: %w", err)
	}
	return nil
}

func mountSandboxDirs() error {
	dirs := strings.Split(os.Getenv(sandboxDirsVar), ",")
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if err := syscall.Mount("", d, "tmpfs", mdLazytime|syscall.MS_NOATIME|syscall.MS_NODEV|syscall.MS_NOSUID, ""); err != nil {
			return fmt.Errorf("Failed to mount sandbox dir %s: %w", d, err)
		}
	}

	return os.Unsetenv(sandboxDirsVar)
}

// loUp brings up the loopback network interface.
func loUp() error {
	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(sock)
	ifreq, err := unix.NewIfreq("lo")
	if err != nil {
		return err
	}
	if err := unix.IoctlIfreq(sock, unix.SIOCGIFFLAGS, ifreq); err != nil {
		return err
	}
	ifreq.SetUint32(ifreq.Uint32() | unix.IFF_UP)
	return unix.IoctlIfreq(sock, unix.SIOCSIFFLAGS, ifreq)
}
