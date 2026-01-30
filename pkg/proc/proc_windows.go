//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"sync"
	"unsafe"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
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
	log.Info("initialized Windows Job Object for process hygiene")

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
	p := metrics.GetProvider()
	// Ensure the job object is created
	jobOnce.Do(initJob)
	if jobErr != nil {
		p.IncProcessFailed()
		return jobErr
	}

	if err := cmd.Start(); err != nil {
		p.IncProcessFailed()
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
		// If we can't open the process to manage it, we must kill it to avoid zombies (fail-closed).
		log.Error("failed to open process for job assignment", "pid", pid, "error", err)
		_ = cmd.Process.Kill()
		return fmt.Errorf("open process: %w", err)
	}
	defer windows.CloseHandle(procHandle)

	if err := windows.AssignProcessToJobObject(jobHandle, procHandle); err != nil {
		// Fail-closed: kill process if we can't guarantee hygiene.
		log.Error("failed to assign process to job object", "pid", pid, "error", err)
		_ = cmd.Process.Kill()
		return fmt.Errorf("assign process to job: %w", err)
	}

	log.Debug("assigned process to job object", "pid", pid)
	p.IncProcessStarted()
	return nil
}
