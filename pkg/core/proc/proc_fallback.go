//go:build !windows && !linux

package proc

import (
	"errors"
	"os/exec"
	"runtime"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
)

func start(cmd *exec.Cmd) error {
	p := metrics.GetProvider()
	// Fallback for macOS, BSD, etc. where Pdeathsig/JobObjects aren't available
	// or implemented yet.
	if StrictMode {
		return errors.New("process hygiene not supported on " + runtime.GOOS)
	}

	log.Warn("process hygiene is not supported on this platform, falling back to standard cmd.Start()",
		"os", runtime.GOOS,
		"arch", runtime.GOARCH)

	err := cmd.Start()
	if err != nil {
		p.IncProcessFailed()
		return err
	}
	p.IncProcessStarted()
	return nil
}
