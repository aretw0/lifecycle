//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	jobHandle windows.Handle
	jobOnce   sync.Once
	jobErr    error
)

func initJob() {
	// Create a Job Object that kills all processes when the handle is closed.
	// Since 'jobHandle' is a global variable that is never explicitly closed,
	// it will be closed when the main process (this process) exits/terminates.
	// That satisfies the requirement: Parent dies -> Child dies.

	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		jobErr = fmt.Errorf("create job object: %w", err)
		return
	}
	jobHandle = h

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}

	if _, err := windows.SetInformationJobObject(
		h,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		jobErr = fmt.Errorf("set job object info: %w", err)
		// Try to close if we failed to set info, to avoid leaking a useless handle
		_ = windows.CloseHandle(h)
		jobHandle = 0
	}
}

func start(cmd *exec.Cmd) error {
	// Ensure the job object is created
	jobOnce.Do(initJob)
	if jobErr != nil {
		return jobErr
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// On Windows, we need to open the process handle with specific permissions
	// to assign it to a job.
	pid := uint32(cmd.Process.Pid)

	// OpenProcess requires PROCESS_SET_QUOTA | PROCESS_TERMINATE to assign to job.
	// We use OpenProcess because cmd.Process doesn't expose the handle directly in a usable way cross-version
	// without unsafe hacks or assuming it's kept open (which it is, but os.Process hides it).
	procHandle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		pid,
	)
	if err != nil {
		// If the process died immediately, we might fail to open it.
		// That's fine, it's not a zombie.
		// To be safe, we can check if it's still running, but usually OpenProcess fails if PID is gone?
		// Actually Windows reuses PIDs, so let's be careful.
		// But cmd.Start just returned, so it's unlikely reused *that* fast unless heavily loaded system.
		// For now, return error, but maybe log it?
		// The requirement is "Start(cmd) error", so returning error is correct behavior if we can't guarantee safety.
		// However, leaving the process running (if we allocated it) but failing to assign logic
		// means we have a potential zombie. Ideally we should kill it if we fail to assign?
		_ = cmd.Process.Kill()
		return fmt.Errorf("open process for job assignment: %w", err)
	}
	defer windows.CloseHandle(procHandle)

	if err := windows.AssignProcessToJobObject(jobHandle, procHandle); err != nil {
		// If assignment fails, we should kill the process to ensure we don't leak it
		// (fail closed).
		_ = cmd.Process.Kill()
		return fmt.Errorf("assign process to job: %w", err)
	}

	return nil
}
