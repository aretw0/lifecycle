//go:build !windows && !linux

package proc

import "os/exec"

func start(cmd *exec.Cmd) error {
	// Fallback for macOS, BSD, etc. where Pdeathsig/JobObjects aren't available
	// or implemented yet.
	return cmd.Start()
}
