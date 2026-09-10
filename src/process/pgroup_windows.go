package process

import (
	"os/exec"

	"golang.org/x/sys/windows"
)

// ShareParentProcessGroup configures a command to run in our process group rather than one of
// its own. Interactive commands need this for stdin and stdout to attach correctly, at the
// cost of no longer being killable as a group.
func ShareParentProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr.CreationFlags &^= windows.CREATE_NEW_PROCESS_GROUP
}
