package proc

import "os/exec"

// StrictMode if true, will cause Start to return an error on unsupported platforms
// instead of just logging a warning. Default is false.
var StrictMode bool

// Start starts the specified command but ensures that the child process
// is killed if the parent process (this process) dies.
//
// On Linux, it uses SysProcAttr.Pdeathsig (SIGKILL).
// On Windows, it uses Job Objects (JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE).
//
// On other platforms, it falls back to cmd.Start() and logs a warning,
// unless StrictMode is set to true, in which case it returns an error.
//
// This is a safer alternative to cmd.Start() for long-running child processes.
func Start(cmd *exec.Cmd) error {
	return start(cmd)
}
