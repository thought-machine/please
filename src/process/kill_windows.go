package process

import (
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows has no process groups that descendants inherit, so killing a whole tree needs a job
// object: every process assigned to one, and everything it subsequently spawns, dies together
// on TerminateJobObject. JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE means that also happens if we exit
// abnormally without cleaning up, which is what Pdeathsig buys us on Linux.
var (
	jobsMux sync.Mutex
	jobs    = map[*exec.Cmd]windows.Handle{}
)

// trackProcessTree assigns a started process to a new job object so that it and its
// descendants can be killed together.
//
// There is an unavoidable race here: the process is already running by the time we assign it,
// so anything it spawns in that window escapes the job. Closing it would need CREATE_SUSPENDED
// and a ResumeThread, which os/exec gives us no way to do.
func trackProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		log.Warning("Failed to create job object, child processes may outlive us: %s", err)
		return
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		log.Warning("Failed to configure job object: %s", err)
		windows.CloseHandle(job)
		return
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		log.Warning("Failed to open process %d: %s", cmd.Process.Pid, err)
		windows.CloseHandle(job)
		return
	}
	defer windows.CloseHandle(proc)
	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		log.Warning("Failed to assign process %d to job object: %s", cmd.Process.Pid, err)
		windows.CloseHandle(job)
		return
	}
	jobsMux.Lock()
	defer jobsMux.Unlock()
	jobs[cmd] = job
}

// untrackProcessTree closes the job object for a command. Because the job is created with
// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE, this also kills anything still running in it.
func untrackProcessTree(cmd *exec.Cmd) {
	jobsMux.Lock()
	job, present := jobs[cmd]
	delete(jobs, cmd)
	jobsMux.Unlock()
	if present {
		windows.CloseHandle(job)
	}
}

// killProcessTree signals a process and all of its descendants.
// SIGTERM is translated to a Ctrl-Break on the console process group, which is the closest
// thing Windows has to a signal a process can handle. It is best-effort: it does not reach
// processes that have detached from the console, and GUI subsystem processes ignore it.
// Anything else terminates the job object, which is unconditional.
func killProcessTree(cmd *exec.Cmd, sig syscall.Signal) error {
	if sig == syscall.SIGTERM {
		if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid)); err == nil {
			return nil
		}
		// Fall through to terminating the job if we couldn't deliver it.
	}
	jobsMux.Lock()
	job, present := jobs[cmd]
	jobsMux.Unlock()
	if !present {
		// No job object, so the best we can do is the process itself.
		return cmd.Process.Kill()
	}
	return windows.TerminateJobObject(job, 1)
}
