//go:build !windows && !linux

package proc

import (
	"errors"
	"os/exec"
	"runtime"

	"github.com/aretw0/lifecycle/pkg/log"
)

func start(cmd *exec.Cmd) error {
	// Fallback for macOS, BSD, etc. where Pdeathsig/JobObjects aren't available
	// or implemented yet.
	if StrictMode {
		return errors.New("process hygiene not supported on " + runtime.GOOS)
	}

	log.Warn("process hygiene is not supported on this platform, falling back to standard cmd.Start()",
		"os", runtime.GOOS,
		"arch", runtime.GOARCH)

	return cmd.Start()
}
