//go:build !windows
// +build !windows

package process

import "os/exec"

// ShareParentProcessGroup configures a command to run in our process group rather than one of
// its own. Interactive commands need this for stdin and stdout to attach correctly, at the
// cost of no longer being killable as a group.
func ShareParentProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr.Setpgid = false
}
