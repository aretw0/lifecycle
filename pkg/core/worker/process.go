// PADRÃO DE SINCRONIZAÇÃO
//
// Utilize withLock para alterações rápidas e atômicas sob o mutex.
// Utilize withLockResult para leituras atômicas que retornam valor.
// Exemplo:
//   valor := withLockResult(p, func() int { return p.meuCampo })
//   withLock(p, func() { p.meuCampo = 42 })
//
// Isso reduz boilerplate, previne unlocks indevidos e facilita manutenção.

package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/aretw0/procio/proc"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/observe"
)

// withLockAny executa uma função sob o mutex de qualquer struct que tenha mu sync.Locker.
type locker interface {
	Lock()
	Unlock()
}

func withLockAny(l locker, fn func()) {
	l.Lock()
	defer l.Unlock()
	fn()
}

// withLockResultAny executa uma função sob o mutex e retorna um valor, para qualquer struct com mu sync.Locker.
func withLockResultAny[T any](l locker, fn func() T) T {
	l.Lock()
	defer l.Unlock()
	return fn()
}

// withLock executa uma função sob o mutex do worker.
// Use para leituras/alterações rápidas e atômicas.
func withLock(p *ProcessWorker, fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn()
}

// withLockResult executa uma função sob o mutex e retorna um valor.
// Útil para leituras atômicas que retornam resultado.
func withLockResult[T any](p *ProcessWorker, fn func() T) T {
	p.mu.Lock()
	defer p.mu.Unlock()
	return fn()
}

// isMutable retorna true se o worker está em estado que permite alteração de configuração.
func (p *ProcessWorker) isMutable() bool {
	return withLockResult(p, func() bool {
		return p.status == StatusCreated || p.status == StatusPending
	})
}

// ProcessWorker is a Worker that manages an OS process.
//
// Concorrência: Todos os métodos que alteram estado interno usam mutex.
// Após o processo ser iniciado, SetEnv e SetOutput não têm efeito e retornam erro.
// O ciclo de vida do processo é Start -> (Stop|Finish) -> State.
// O método Stop pode retornar múltiplos erros combinados via errors.Join (veja documentação).
//
// State.Metadata inclui os campos "startedAt" e "stoppedAt" (RFC3339Nano) para rastreabilidade.
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
	if !p.isMutable() {
		return fmt.Errorf("worker %s already started (status: %s)", p.String(), p.status)
	}

	log.Info("starting process worker", "name", p.String(), "cmd", p.cmd.Path, "args", p.cmd.Args)

	withLock(p, func() {
		// Inject environment variables
		if len(p.env) > 0 {
			if p.cmd.Env == nil {
				p.cmd.Env = p.cmd.Environ()
			}
			for k, v := range p.env {
				p.cmd.Env = append(p.cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
	})

	// Use pkg/proc to start with hygiene guarantees
	if err := proc.Start(p.cmd); err != nil {
		if obs := observe.GetObserver(); obs != nil {
			obs.OnProcessFailed(err)
		}
		withLock(p, func() { p.Err = err })
		p.SetStatus(StatusFailed)
		metrics.GetProvider().IncWorkerFailed("process")
		return fmt.Errorf("failed to start worker %s: %w", p.String(), err)
	}

	withLock(p, func() { p.startedAt = time.Now() })
	if obs := observe.GetObserver(); obs != nil && p.cmd.Process != nil {
		obs.OnProcessStarted(p.cmd.Process.Pid)
	}

	p.SetStatus(StatusRunning)
	metrics.GetProvider().IncWorkerStarted("process")
	log.Info("process worker started", "name", p.String(), "cmd", p.cmd.Path)

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

	// First try graceful storage (SIGINT/SIGTERM depending on platform/app)
	// On Windows, os.Interrupt é o único sinal capturável via os.Process.Signal
	sigErr := process.Signal(os.Interrupt)

	// Wait for quiescence or timeout using Base implementation
	err := p.BaseWorker.Stop(ctx)
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
		withLock(p, func() { p.Killed = true })
		log.Warn("force killing process worker", "name", p.String(), "pid", process.Pid)
		killErr := process.Kill()
		// Retorna todos os erros relevantes para o chamador analisar.
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
		if p.cmd.Process != nil {
			s.PID = p.cmd.Process.Pid
		}
		s.ExitCode = p.ExitCode
		s.Error = p.Err
		s.Metadata = map[string]string{
			"type":      "process",
			"path":      p.cmd.Path,
			"args":      fmt.Sprintf("%v", p.cmd.Args),
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
			p.cmd.Stdout = stdout
			p.cmd.Stderr = stderr
		})
	}
	if !allowed {
		return fmt.Errorf("SetOutput só pode ser chamado antes de Start")
	}
	return nil
}
