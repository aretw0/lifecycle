package proc

import "os/exec"

// Start starts the specified command but ensures that the child process
// is killed if the parent process (this process) dies.
//
// On Linux, it uses SysProcAttr.Pdeathsig (SIGKILL).
// On Windows, it uses Job Objects (JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE).
//
// This is a safer alternative to cmd.Start() for long-running child processes.
func Start(cmd *exec.Cmd) error {
	return start(cmd)
}
