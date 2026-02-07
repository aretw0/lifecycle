package worker

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/proc"
)

// ProcessWorker is a Worker that manages an OS process.
type ProcessWorker struct {
	*BaseWorker
	cmd *exec.Cmd

	startedAt time.Time
	stoppedAt time.Time
	exitCode  int
	err       error
	env       map[string]string
}

// NewProcessWorker creates a new ProcessWorker for the given command.
func NewProcessWorker(name string, nameCmd string, args ...string) *ProcessWorker {
	cmd := exec.Command(nameCmd, args...)

	return &ProcessWorker{
		BaseWorker: NewBaseWorker(name),
		cmd:        cmd,
		env:        make(map[string]string),
	}
}

// Start starts the OS process.
func (p *ProcessWorker) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.status != StatusCreated && p.status != StatusPending {
		p.mu.Unlock()
		return fmt.Errorf("worker %s already started (status: %s)", p.String(), p.status)
	}

	log.Info("starting process worker", "name", p.String(), "cmd", p.cmd.Path, "args", p.cmd.Args)

	// Inject environment variables
	for k, v := range p.env {
		p.cmd.Env = append(p.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Use pkg/proc to start with hygiene guarantees
	if err := proc.Start(p.cmd); err != nil {
		p.status = StatusFailed
		p.err = err
		p.mu.Unlock()
		p.emitStateChange(State{Name: p.String(), Status: StatusCreated}, State{Name: p.String(), Status: StatusFailed})
		metrics.GetProvider().IncWorkerFailed("process")
		return fmt.Errorf("failed to start worker %s: %w", p.String(), err)
	}

	p.status = StatusRunning
	p.startedAt = time.Now()
	p.mu.Unlock()
	p.emitStateChange(State{Name: p.String(), Status: StatusCreated}, State{Name: p.String(), Status: StatusRunning})
	metrics.GetProvider().IncWorkerStarted("process")

	// Monitor in background
	go func() {
		err := p.cmd.Wait()

		p.mu.Lock()
		p.stoppedAt = time.Now()
		duration := p.stoppedAt.Sub(p.startedAt)
		p.err = err
		oldStatus := p.status
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
		newStatus := p.status
		p.mu.Unlock()

		p.emitStateChange(State{Name: p.String(), Status: oldStatus}, State{Name: p.String(), Status: newStatus})
		metrics.GetProvider().ObserveWorkerDuration("process", duration)
		log.Info("process worker stopped", "name", p.String(), "exit_code", p.exitCode, "error", err, "duration", duration)

		p.done <- err
		close(p.done)
	}()

	return nil
}

// Stop sends a signal to the process to terminate.
func (p *ProcessWorker) Stop(ctx context.Context) error {
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

	log.Debug("stopping process worker", "name", p.String(), "pid", process.Pid)
	stopStart := time.Now()

	// First try graceful storage (SIGINT/SIGTERM depending on platform/app)
	_ = process.Signal(syscall.SIGTERM)

	// Wait for exit or context timeout (Force Kill)
	select {
	case <-p.done:
		metrics.GetProvider().ObserveShutdownDuration("process", time.Since(stopStart))
		return nil
	case <-ctx.Done():
		// Context expired, force kill
		log.Warn("force killing process worker", "name", p.String(), "pid", process.Pid)
		_ = process.Kill()
		metrics.GetProvider().ObserveShutdownDuration("process", time.Since(stopStart))
		return ctx.Err()
	}
}

// String returns the worker name.
func (p *ProcessWorker) String() string {
	return fmt.Sprintf("Process(%s)", p.BaseWorker.String())
}

// SetEnv adds an environment variable to the process.
func (p *ProcessWorker) SetEnv(key, value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.env[key] = value
}

// State returns a snapshot of the worker's status.
func (p *ProcessWorker) State() State {
	return p.ExportState(func(s *State) {
		if p.cmd.Process != nil {
			s.PID = p.cmd.Process.Pid
		}
		s.ExitCode = p.exitCode
		s.Error = p.err
		s.Metadata = map[string]string{
			"type": "process",
			"path": p.cmd.Path,
			"args": fmt.Sprintf("%v", p.cmd.Args),
		}
	})
}
