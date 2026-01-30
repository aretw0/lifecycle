package worker

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"

	"github.com/aretw0/lifecycle/pkg/proc"
)

// Process is a Worker that manages an OS process.
type Process struct {
	cmd  *exec.Cmd
	name string

	mu       sync.Mutex
	started  bool
	waitChan chan error
}

// NewProcess creates a new Process worker for the given command.
func NewProcess(name string, nameCmd string, args ...string) *Process {
	cmd := exec.Command(nameCmd, args...)
	// proc.Start will handle hygiene (JobObjects/PDeathSig)

	return &Process{
		cmd:      cmd,
		name:     name,
		waitChan: make(chan error, 1),
	}
}

// Start starts the OS process.
func (p *Process) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("worker %s already started", p.name)
	}

	// Use pkg/proc to start with hygiene guarantees
	if err := proc.Start(p.cmd); err != nil {
		return fmt.Errorf("failed to start worker %s: %w", p.name, err)
	}

	p.started = true

	// Monitor in background
	go func() {
		err := p.cmd.Wait()
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
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	process := p.cmd.Process
	p.mu.Unlock()

	if process == nil {
		return nil
	}

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
		return nil
	case <-ctx.Done():
		// Context expired, force kill
		_ = process.Kill()
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
