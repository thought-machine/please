//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// mdLazytime is the bit for lazily flushing disk writes.
// TODO(jpoole): find out if there's a reason this isn't in syscall
const mdLazytime = 1 << 25

const sandboxDirsVar = "SANDBOX_DIRS"

// Avoid importing this from Please. This tool should initialise quickly.
var sandboxMountDir = "/tmp/plz_sandbox"

func sandbox(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("incorrect number of args to call plz sandbox")
	}

	env := os.Environ()
	cmd, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("Failed to lookup %s on path: %s", args[0], err)
	}

	unshareMount := os.Getenv("SHARE_MOUNT") != "1"
	unshareNetwork := os.Getenv("SHARE_NETWORK") != "1"
	user := os.Getenv("SANDBOX_UID")

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

		rewriteEnvVars(env, tmpDirEnv, sandboxMountDir)

		if err := os.Chdir(sandboxMountDir); err != nil {
			return fmt.Errorf("Failed to chdir to %s: %s", sandboxMountDir, err)
		}

		if err := mountProc(); err != nil {
			return err
		}
	}

	if unshareNetwork {
		if err := loUp(); err != nil {
			return fmt.Errorf("Failed to bring loopback interface up: %s", err)
		}
	}

	if user != "" {
		userID, err := strconv.Atoi(user)
		if err != nil {
			return fmt.Errorf("invalid SANDBOX_UID: %v", user)
		}
		execCmd := exec.Command(cmd, args[1:]...)
		execCmd.Env = env
		execCmd.Stdout = os.Stdout
		execCmd.Stdin = os.Stdin
		execCmd.Stderr = os.Stderr
		execCmd.SysProcAttr = &syscall.SysProcAttr{
			Pdeathsig:  syscall.SIGHUP,
			Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID,
			UidMappings: []syscall.SysProcIDMap{
				{HostID: os.Getuid(), Size: 1, ContainerID: userID},
			},
			GidMappings: []syscall.SysProcIDMap{
				{HostID: os.Getgid(), Size: 1, ContainerID: userID},
			},
		}
		return execCmd.Run()
	}
	err = syscall.Exec(cmd, args, env)
	if err != nil {
		return fmt.Errorf("Failed to exec %s: %s", cmd, err)
	}
	return nil
}

func rewriteEnvVars(env []string, from, to string) {
	for i, envVar := range env {
		if strings.Contains(envVar, from) {
			parts := strings.Split(envVar, "=")
			key := parts[0]
			value := strings.TrimPrefix(envVar, key+"=")
			env[i] = key + "=" + strings.ReplaceAll(value, from, to)
		}
	}
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

func main() {
	if err := sandbox(os.Args[1:]); err != nil {
		fmt.Printf("Failed to run sandbox command: %v", err)
		os.Exit(1)
	}
}
