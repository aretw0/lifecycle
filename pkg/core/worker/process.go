package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aretw0/procio/proc"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/observe"
)

// ProcessWorker is a Worker that manages an OS process.
//
// Concorrência: Todos os métodos que alteram estado interno usam mutex.
// Após o processo ser iniciado, SetEnv e SetOutput não têm efeito e retornam erro.
// O ciclo de vida do processo é Start -> (Stop|Finish) -> State.
// O método Stop pode retornar múltiplos erros combinados via errors.Join.
//
// State.Metadata inclui os campos "startedAt" e "stoppedAt" (RFC3339Nano) para rastreabilidade.
type ProcessWorker struct {
	*BaseWorker
	nameCmd string
	args    []string
	cmd     *proc.Cmd

	startedAt time.Time
	stoppedAt time.Time
	env       map[string]string
	stdout    io.Writer
	stderr    io.Writer
}

// NewProcessWorker creates a new ProcessWorker for the given command.
func NewProcessWorker(name string, nameCmd string, args ...string) *ProcessWorker {
	return &ProcessWorker{
		BaseWorker: NewBaseWorker(name),
		nameCmd:    nameCmd,
		args:       args,
		env:        make(map[string]string),
	}
}

// isMutable retorna true se o worker está em estado que permite alteração de configuração.
func (p *ProcessWorker) isMutable() bool {
	return withLockResult(p, func() bool {
		return p.status == StatusCreated || p.status == StatusPending
	})
}

// Start initiates the OS process.
func (p *ProcessWorker) Start(ctx context.Context) error {
	p.SetStatus(StatusStarting)

	// Lazy command construction ensures context is linked and hygiene is applied.
	p.cmd = proc.NewCmd(ctx, p.nameCmd, p.args...)

	withLock(p, func() {
		p.cmd.Stdout = p.stdout
		p.cmd.Stderr = p.stderr

		// Inject environment variables
		if len(p.env) > 0 {
			p.cmd.Env = p.cmd.Environ()
			for k, v := range p.env {
				p.cmd.Env = append(p.cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
	})

	// Start with hygiene guarantees already configured by NewCmd
	if err := p.cmd.Start(); err != nil {
		if obs := observe.GetObserver(); obs != nil {
			obs.OnProcessFailed(err)
		}
		// Failure path: ensure Wait() channels are closed and status is terminal.
		p.Finish(err)
		metrics.GetProvider().IncWorkerFailed("process")
		return fmt.Errorf("failed to start worker %s: %w", p.String(), err)
	}

	withLock(p, func() { p.startedAt = time.Now() })
	if obs := observe.GetObserver(); obs != nil && p.cmd.Process != nil {
		obs.OnProcessStarted(p.cmd.Process.Pid)
	}

	p.SetStatus(StatusRunning)
	metrics.GetProvider().IncWorkerStarted("process")
	log.Info("process worker started", "name", p.String(), "pid", p.cmd.Process.Pid)

	// Monitor in background
	go func() {
		err := p.cmd.Wait()

		result := withLockResult(p, func() struct {
			duration time.Duration
			exitCode int
		} {
			p.stoppedAt = time.Now()
			duration := p.stoppedAt.Sub(p.startedAt)
			exitCode := -1
			if p.cmd.ProcessState != nil {
				exitCode = p.cmd.ProcessState.ExitCode()
			}
			p.ExitCode = exitCode
			return struct {
				duration time.Duration
				exitCode int
			}{duration, exitCode}
		})

		p.Finish(err)
		metrics.GetProvider().ObserveWorkerDuration("process", result.duration)
		log.Info("process worker stopped", "name", p.String(), "exit_code", result.exitCode, "error", err, "duration", result.duration)
	}()

	return nil
}

// Stop sends a signal to the process to terminate.
// Pode retornar erro composto (errors.Join) contendo erros de sinalização e kill.
// Use errors.Is/As para inspecionar causas individuais.
func (p *ProcessWorker) Stop(ctx context.Context) error {
	notRunning := withLockResult(p, func() bool {
		if p.status != StatusRunning {
			return true
		}
		p.StopRequested = true
		return false
	})
	if notRunning {
		return nil
	}

	process := p.cmd.Process
	if process == nil {
		return nil
	}

	log.Debug("stopping process worker", "name", p.String(), "pid", process.Pid)

	// First try graceful shutdown (SIGINT/SIGTERM depending on platform/app)
	// On Windows, os.Interrupt is the only catchable signal via os.Process.Signal
	sigErr := p.cmd.Process.Signal(os.Interrupt)

	// Wait for quiescence or timeout using Base implementation
	err := p.BaseWorker.Stop(ctx)
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
		withLock(p, func() { p.Killed = true })
		log.Warn("force killing process worker", "name", p.String(), "pid", p.cmd.Process.Pid)
		killErr := p.cmd.Process.Kill()
		// Return joined errors for inspection
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
// Só pode ser chamado antes de Start. Após o início, retorna erro.
func (p *ProcessWorker) SetEnv(key, value string) error {
	allowed := p.isMutable()
	if allowed {
		withLock(p, func() { p.env[key] = value })
	}
	if !allowed {
		return fmt.Errorf("SetEnv só pode ser chamado antes de Start")
	}
	return nil
}

// State returns a snapshot of the worker's status, incluindo timestamps.
func (p *ProcessWorker) State() State {
	// Não usar withLockResult aqui, pois ExportState já faz lock internamente.
	return p.ExportState(func(s *State) {
		if p.cmd != nil && p.cmd.Process != nil {
			s.PID = p.cmd.Process.Pid
		}
		s.ExitCode = p.ExitCode
		s.Error = p.Err
		s.Metadata = map[string]string{
			"type":      "process",
			"path":      p.nameCmd,
			"args":      fmt.Sprintf("%v", p.args),
			"startedAt": p.startedAt.Format(time.RFC3339Nano),
			"stoppedAt": p.stoppedAt.Format(time.RFC3339Nano),
		}
	})
}

// SetOutput configures the standard output and error writers for the process.
// Deve ser chamado antes de Start. Após o início, retorna erro.
func (p *ProcessWorker) SetOutput(stdout, stderr io.Writer) error {
	allowed := p.isMutable()
	if allowed {
		withLock(p, func() {
			p.stdout = stdout
			p.stderr = stderr
		})
	}
	if !allowed {
		return fmt.Errorf("SetOutput só pode ser chamado antes de Start")
	}
	return nil
}
