package worker

import (
	"context"
	"errors"
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
		p.Err = err
		p.mu.Unlock()

		p.SetStatus(StatusFailed)
		metrics.GetProvider().IncWorkerFailed("process")
		return fmt.Errorf("failed to start worker %s: %w", p.String(), err)
	}

	p.startedAt = time.Now()
	p.mu.Unlock()

	p.SetStatus(StatusRunning)
	metrics.GetProvider().IncWorkerStarted("process")
	log.Info("process worker started", "name", p.String(), "cmd", p.cmd.Path)

	// Monitor in background
	go func() {
		err := p.cmd.Wait()

		p.mu.Lock()
		p.stoppedAt = time.Now()
		duration := p.stoppedAt.Sub(p.startedAt)

		// Capture results for metrics only
		exitCode := -1
		if p.cmd.ProcessState != nil {
			exitCode = p.cmd.ProcessState.ExitCode()
		}

		p.ExitCode = exitCode
		p.mu.Unlock()

		p.Finish(err)
		metrics.GetProvider().ObserveWorkerDuration("process", duration)
		log.Info("process worker stopped", "name", p.String(), "exit_code", p.ExitCode, "error", err, "duration", duration)
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
	p.StopRequested = true
	p.mu.Unlock()

	if process == nil {
		return nil
	}

	log.Debug("stopping process worker", "name", p.String(), "pid", process.Pid)

	// First try graceful storage (SIGINT/SIGTERM depending on platform/app)
	sigErr := process.Signal(syscall.SIGTERM)

	// Wait for quiescence or timeout using Base implementation
	err := p.BaseWorker.Stop(ctx)
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
		p.mu.Lock()
		p.Killed = true
		p.mu.Unlock()

		log.Warn("force killing process worker", "name", p.String(), "pid", process.Pid)
		killErr := process.Kill()
		return errors.Join(err, sigErr, killErr)
	}

	if err != nil {
		return errors.Join(err, sigErr)
	}

	return nil
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
		s.ExitCode = p.ExitCode
		s.Error = p.Err
		s.Metadata = map[string]string{
			"type": "process",
			"path": p.cmd.Path,
			"args": fmt.Sprintf("%v", p.cmd.Args),
		}
	})
}
