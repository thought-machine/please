// +build linux

package sandbox

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
	"os"
	"os/exec"
	"strings"
	"syscall"

	// #include "sandbox.h"
	"C"
)

// mdLazytime is the bit for lazily flushing disk writes.
// TODO(jpoole): find out if there's a reason this isn't in syscall
const mdLazytime = 1 << 25

const sandboxDirsVar = "SANDBOX_DIRS"

func Sandbox(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("incorrect number of args to call plz sandbox")
	}
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	tmpDir := os.Getenv("TMP_DIR")
	if err := mountTmp(tmpDir); err != nil {
		return err
	}

	if err := mountProc(); err != nil {
		return err
	}

	if i := C.lo_up(); i < 0 {
		return fmt.Errorf("failed to bring loopback interface up")
	}

	if err := mountSandboxDirs(); err != nil {
		return fmt.Errorf("failed to mount over sandboxed dirs: %w", err)
	}

	if tmpDir != "" {
		cmd.Dir = core.SandboxDir
		if err := rewriteEnvVars(tmpDir); err != nil {
			return err
		}
	}

	return cmd.Run()
}

func rewriteEnvVars(tmpDir string) error {
	for _, envVar := range os.Environ() {
		if strings.Contains(envVar, tmpDir) {
			parts := strings.Split(envVar, "=")
			key := parts[0]
			value := strings.TrimPrefix(envVar, key+"=")
			if err := os.Setenv(key, strings.ReplaceAll(value, tmpDir, core.SandboxDir)); err != nil {
				return err
			}
		}
	}
	return nil
}

func mountTmp(tmpDir string) error {
	dir := core.SandboxDir

	if strings.HasPrefix(tmpDir, "/tmp") {
		_, err := fmt.Fprintln(os.Stderr, "Not mounting /tmp as $TMP_DIR is a subdir")
		return err
	}

	// Remounting / as private is necessary so that the tmpfs mount isn't visible to anyone else.
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to mount root: %w", err)
	}

	flags := mdLazytime | syscall.MS_NOATIME | syscall.MS_NODEV | syscall.MS_NOSUID
	if err := syscall.Mount("", "/tmp", "tmpfs", uintptr(flags), ""); err != nil {
		return fmt.Errorf("failed to mount /tmp: %w", err)
	}

	if err := os.Setenv("TMPDIR", "/tmp"); err != nil {
		return fmt.Errorf("failed to set $TMPDIR: %w", err)
	}

	if tmpDir == "" {
		_, err := fmt.Fprintln(os.Stderr, "$TMP_DIR not set, will not bind-mount to ", core.SandboxDir)
		return err
	}

	if err := os.Mkdir(dir, os.ModeDir|0775); err != nil {
		return fmt.Errorf("failed to make %s: %w", dir, err)
	}

	if err := syscall.Mount(tmpDir, dir, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to bind %s to %s : %w", tmpDir, dir, err)
	}

	if err := syscall.Mount("", "/", "", syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to remount root as readonly: %s", err)
	}
	return nil
}

func mountProc() error {
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount /proc: %w", err)
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
			return fmt.Errorf("failed to mount sandbox dir %s: %w", d, err)
		}
	}

	return os.Unsetenv(sandboxDirsVar)
}
