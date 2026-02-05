// Package proc provides primitives for managing process lifecycle and hygiene.
//
// Its primary goal is to prevent "zombie" processes by ensuring that child processes
// spawned by the application are automatically terminated when the parent process exits
// (concept known as "death pact" or "process hygiene").
//
// # Mechanisms
//
//   - Windows: Uses Job Objects with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE.
//   - Linux: Uses SysProcAttr.Pdeathsig with SIGKILL.
//   - Others: Falls back to standard os/exec behavior (no guarantee of cleanup).
//
// # Usage
//
// Replace exec.Cmd.Start() with proc.Start(cmd):
//
//	cmd := exec.Command("worker")
//	if err := proc.Start(cmd); err != nil {
//	    log.Fatal(err)
//	}
//
// The child process is now linked to the parent's lifecycle.
package proc



