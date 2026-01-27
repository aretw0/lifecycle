//go:build linux

package proc

import (
	"os/exec"
	"syscall"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

func start(cmd *exec.Cmd) error {
	p := metrics.GetProvider()
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Pdeathsig ensures that if the parent dies, the child receives this signal.
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL

	log.Debug("starting process with Pdeathsig", "command", cmd.Path)
	err := cmd.Start()
	if err != nil {
		p.IncProcessFailed()
		return err
	}
	p.IncProcessStarted()
	return nil
}
