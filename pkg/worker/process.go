package worker

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/proc"
)

// Process is a Worker that manages an OS process.
type Process struct {
	cmd  *exec.Cmd
	name string

	mu        sync.Mutex
	status    Status
	startedAt time.Time
	stoppedAt time.Time
	exitCode  int
	err       error
	waitChan  chan error
}

// NewProcess creates a new Process worker for the given command.
func NewProcess(name string, nameCmd string, args ...string) *Process {
	cmd := exec.Command(nameCmd, args...)
	// proc.Start will handle hygiene (JobObjects/PDeathSig)

	return &Process{
		cmd:      cmd,
		name:     name,
		status:   StatusPending,
		waitChan: make(chan error, 1),
	}
}

// Start starts the OS process.
func (p *Process) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != StatusPending {
		return fmt.Errorf("worker %s already started (status: %s)", p.name, p.status)
	}

	log.Info("starting process worker", "name", p.name, "cmd", p.cmd.Path, "args", p.cmd.Args)

	// Use pkg/proc to start with hygiene guarantees
	if err := proc.Start(p.cmd); err != nil {
		p.status = StatusFailed
		p.err = err
		metrics.GetProvider().IncWorkerFailed("process")
		return fmt.Errorf("failed to start worker %s: %w", p.name, err)
	}

	p.status = StatusRunning
	p.startedAt = time.Now()
	metrics.GetProvider().IncWorkerStarted("process")

	// Monitor in background
	go func() {
		err := p.cmd.Wait()

		p.mu.Lock()
		p.stoppedAt = time.Now()
		duration := p.stoppedAt.Sub(p.startedAt)
		p.err = err
		if err != nil {
			// Try to extract exit code
			if exitErr, ok := err.(*exec.ExitError); ok {
				p.exitCode = exitErr.ExitCode()
			} else {
				p.exitCode = -1
			}
			p.status = StatusFailed
			metrics.GetProvider().IncWorkerFailed("process")
		} else {
			p.exitCode = 0
			p.status = StatusStopped
			metrics.GetProvider().IncWorkerStopped("process")
		}
		p.mu.Unlock()

		metrics.GetProvider().ObserveWorkerDuration("process", duration)
		log.Info("process worker stopped", "name", p.name, "exit_code", p.exitCode, "error", err, "duration", duration)

		if err != nil {
			// If process exits with status code != 0, Wait returns error
			p.waitChan <- err
		} else {
			p.waitChan <- nil
		}
		close(p.waitChan)
	}()

	return nil
}

// Stop sends a signal to the process to terminate.
func (p *Process) Stop(ctx context.Context) error {
	p.mu.Lock()
	if p.status != StatusRunning {
		p.mu.Unlock()
		return nil
	}
	process := p.cmd.Process
	p.mu.Unlock()

	if process == nil {
		return nil
	}

	log.Debug("stopping process worker", "name", p.name, "pid", process.Pid)
	stopStart := time.Now()

	// First try graceful storage (SIGINT/SIGTERM depending on platform/app)
	// For generic processes, SIGTERM is standard.
	// Windows note: Signal won't work same way, but os.Process.Signal handles some cases.
	// Using proc library or os.Process.Signal.
	_ = process.Signal(syscall.SIGTERM)
	// We ignore the error because on Windows SIGTERM is not supported.
	// We will wait for the process to exit (if it handles the signal or if it was the signal that stopped it)
	// or eventually kill it when ctx expires.

	// Wait for exit or context timeout (Force Kill)
	select {
	case <-p.waitChan:
		metrics.GetProvider().ObserveShutdownDuration("process", time.Since(stopStart))
		return nil
	case <-ctx.Done():
		// Context expired, force kill
		log.Warn("force killing process worker", "name", p.name, "pid", process.Pid)
		_ = process.Kill()
		metrics.GetProvider().ObserveShutdownDuration("process", time.Since(stopStart))
		return ctx.Err()
	}
}

// Wait returns the channel to wait for process exit.
func (p *Process) Wait() <-chan error {
	return p.waitChan
}

// String returns the worker name.
func (p *Process) String() string {
	return fmt.Sprintf("Process(%s)", p.name)
}

// State returns a snapshot of the worker's status.
func (p *Process) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()

	pid := 0
	if p.cmd.Process != nil {
		pid = p.cmd.Process.Pid
	}

	return State{
		Name:     p.name,
		Status:   p.status,
		PID:      pid,
		ExitCode: p.exitCode,
		Error:    p.err,
	}
}
