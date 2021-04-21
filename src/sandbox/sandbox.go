package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	// #include "sandbox.h"
	"C"
)

const MS_LAZYTIME = 1 << 25

func Sandbox(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Dir = "/tmp/plz_sandbox"
	if err := mountTmp(); err != nil {
		return err
	}
	if i := C.lo_up(); i < 0 {
		return fmt.Errorf("failed to bring loopback interface up")
	}

	fmt.Println("Welcome to the sandbox!")
	return cmd.Run()
}

func mountTmp() error {
	tmpDir := os.Getenv("TMP_DIR")
	dir := "/tmp/plz_sandbox"

	if strings.HasPrefix(tmpDir, "/tmp") {
		_, err := fmt.Fprintln(os.Stderr, "Not mounting /tmp as $TMP_DIR is a subdir")
		return err
	}

	// Remounting / as private is necessary so that the tmpfs mount isn't visible to anyone else.
	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to mount root: %w", err)
	}

	flags := MS_LAZYTIME | syscall.MS_NOATIME | syscall.MS_NODEV | syscall.MS_NOSUID
	if err := syscall.Mount("tmpfs", "/tmp", "tmpfs", uintptr(flags), ""); err != nil {
		return fmt.Errorf("failed to mount /tmp: %w", err)
	}

	if err := os.Setenv("TMPDIR", "/tmp"); err != nil {
		return fmt.Errorf("failed to set $TMPDIR: %w", err)
	}

	if tmpDir == "" {
		_, err := fmt.Fprintln(os.Stderr, "$TMP_DIR not set, will not bind-mount to /tmp/plz_sandbox")
		return err
	}

	if err := os.Mkdir(dir, os.ModeDir | 0775); err != nil {
		return fmt.Errorf("failed to make %s: %w", dir, err)
	}

	if err := syscall.Mount(tmpDir, dir, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to bind %s to %s : %w", tmpDir, dir, err)
	}

	if err := syscall.Mount("none", "/", "", syscall.MS_REMOUNT | syscall.MS_RDONLY | syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to remount root as readonly: %s", err)
	}
	return nil
}