package lifecycle

import (
	"context"
	"io"
	"os/exec"
	"time"

	"log/slog"

	"github.com/aretw0/lifecycle/pkg/container"
	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/proc"
	"github.com/aretw0/lifecycle/pkg/runtime"
	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/supervisor"
	"github.com/aretw0/lifecycle/pkg/termio"
	"github.com/aretw0/lifecycle/pkg/worker"
)

// NewSignalContext creates a context that cancels on SIGTERM/SIGINT.
// On the first signal, context is cancelled. On the second, it force exits.
// Behavior can be customized via functional options.
// Alias for pkg/signal.NewContext.
func NewSignalContext(parent context.Context, opts ...signal.Option) *signal.Context {
	return signal.NewContext(parent, opts...)
}

// WithInterrupt configures whether SIGINT (Ctrl+C) should cancel the context.
// Alias for pkg/signal.WithInterrupt.
func WithInterrupt(cancel bool) signal.Option {
	return signal.WithInterrupt(cancel)
}

// WithForceExit configures the threshold of signals required to trigger an immediate os.Exit(1).
// Set to 0 to disable forced exit. Alias for pkg/signal.WithForceExit.
func WithForceExit(threshold int) signal.Option {
	return signal.WithForceExit(threshold)
}

// WithHookTimeout configures the duration after which a running hook produces a warning log.
// Alias for pkg/signal.WithHookTimeout.
func WithHookTimeout(d time.Duration) signal.Option {
	return signal.WithHookTimeout(d)
}

// OpenTerminal checks for text input capability and returns a Reader.
// On Windows, it tries to open CONIN$. Alias for pkg/termio.Open.
func OpenTerminal() (io.ReadCloser, error) {
	return termio.Open()
}

// NewInterruptibleReader returns a reader that checks the cancel channel before/after blocking reads.
// Alias for pkg/termio.NewInterruptibleReader.
func NewInterruptibleReader(base io.Reader, cancel <-chan struct{}) *termio.InterruptibleReader {
	return termio.NewInterruptibleReader(base, cancel)
}

// IsInterrupted checks if an error indicates an interruption (Context Canceled, EOF, etc.).
// Alias for pkg/termio.IsInterrupted.
func IsInterrupted(err error) bool {
	return termio.IsInterrupted(err)
}

// UpgradeTerminal checks if the provided reader is a terminal and returns a safe reader (e.g. CONIN$ on Windows).
// If not a terminal, returns the original reader.
func UpgradeTerminal(r io.Reader) (io.Reader, error) {
	return termio.Upgrade(r)
}

// BlockWithTimeout blocks until the done channel is closed or the timeout expires.
// Alias for pkg/runtime.BlockWithTimeout.
func BlockWithTimeout(done <-chan struct{}, timeout time.Duration) error {
	return runtime.BlockWithTimeout(done, timeout)
}

// StartProcess starts the specified command with process hygiene (auto-kill on parent exit).
// Alias for pkg/proc.Start.
func StartProcess(cmd *exec.Cmd) error {
	return proc.Start(cmd)
}

// SetStrictMode sets whether to block on unsupported platforms for process hygiene.
// Alias for pkg/proc.StrictMode.
func SetStrictMode(strict bool) {
	proc.StrictMode = strict
}

// SetLogger overrides the global logger used by the library.
// Alias for pkg/log.SetLogger.
func SetLogger(l *slog.Logger) {
	log.SetLogger(l)
}

// SetMetricsProvider overrides the global metrics provider.
// This allowing bridging library metrics to Prometheus, OTEL, etc.
// Alias for pkg/metrics.SetProvider.
func SetMetricsProvider(p metrics.Provider) {
	metrics.SetProvider(p)
}

// NewLogMetricsProvider returns a metrics provider that logs to the current logger.
// Useful for development and local verification.
// Alias for pkg/metrics.LogProvider.
func NewLogMetricsProvider() metrics.Provider {
	return &metrics.LogProvider{}
}

// Context represents the signal context.
type Context = signal.Context

// Option is a functional option for signal configuration.
type Option = signal.Option

// SignalState represents the configuration state of the SignalContext.
type SignalState = signal.State

// State is an alias for SignalState (backward compatibility).
type State = SignalState

// SignalStateDiagram returns a Mermaid state diagram string representing the signal context configuration.
// Alias for pkg/signal.MermaidState.
func SignalStateDiagram(s SignalState) string {
	return signal.MermaidState(s)
}

// WorkerState represents the snapshot of a worker's state.
type WorkerState = worker.State

// WorkerStatus represents the lifecycle state of a worker.
type WorkerStatus = worker.Status

const (
	WorkerStatusPending = worker.StatusPending
	WorkerStatusRunning = worker.StatusRunning
	WorkerStatusStopped = worker.StatusStopped
	WorkerStatusFailed  = worker.StatusFailed
)

// WorkerTreeDiagram returns a Mermaid diagram string representing the worker structure (Tree).
// Alias for pkg/worker.MermaidTree.
func WorkerTreeDiagram(s WorkerState) string {
	return worker.MermaidTree(s)
}

// WorkerStateDiagram returns a Mermaid state diagram string representing the worker state transitions.
// Alias for pkg/worker.MermaidState.
func WorkerStateDiagram(s WorkerState) string {
	return worker.MermaidState(s)
}

// Worker defines the interface for a managed unit of work.
// Alias for pkg/worker.Worker.
type Worker = worker.Worker

// NewProcessWorker creates a new Process worker for the given command.
// Alias for pkg/worker.NewProcessWorker.
func NewProcessWorker(name string, nameCmd string, args ...string) Worker {
	return worker.NewProcessWorker(name, nameCmd, args...)
}

// NewWorkerFromFunc creates a Worker from a function.
// Alias for pkg/worker.FromFunc.
func NewWorkerFromFunc(name string, fn func(context.Context) error) Worker {
	return worker.FromFunc(name, fn)
}

// Container represents a generic container interface.
// Alias for container.Container.
type Container = container.Container

// ContainerStatus represents the lifecycle state of a container.
// Alias for container.Status.
type ContainerStatus = container.Status

// NewContainerWorker creates a new Worker from a Container interface.
// Alias for pkg/worker.NewContainerWorker.
func NewContainerWorker(name string, c Container) Worker {
	return worker.NewContainerWorker(name, c)
}

// NewMockContainer creates a new MockContainer for testing.
// Alias for pkg/container.NewMockContainer.
func NewMockContainer(id string) Container {
	return container.NewMockContainer(id)
}

// Handover Constants
const (
	// EnvResumeID is the unique session identifier for a worker.
	EnvResumeID = worker.EnvResumeID
	// EnvPrevExit is the exit code of the previous execution of this worker.
	EnvPrevExit = worker.EnvPrevExit
)

// Supervisor defines the interface for a supervisor.
// Alias for pkg/supervisor.Supervisor.
type Supervisor = supervisor.Supervisor

// SupervisorStrategy defines how the supervisor handles child failures.
// Alias for pkg/supervisor.Strategy.
type SupervisorStrategy = supervisor.Strategy

const (
	// StrategyOneForOne: If a child process terminates, only that process is restarted.
	StrategyOneForOne = supervisor.StrategyOneForOne
	// StrategyOneForAll: If a child process terminates, all other child processes are terminated.
	StrategyOneForAll = supervisor.StrategyOneForAll
)

// SupervisorSpec defines the configuration for a supervised child worker.
// Alias for pkg/supervisor.Spec.
type SupervisorSpec = supervisor.Spec

// SupervisorBackoff defines the retry policy for failed children.
// Alias for pkg/supervisor.Backoff.
type SupervisorBackoff = supervisor.Backoff

// SupervisorFactory is a function that creates a new worker instance.
// Alias for pkg/supervisor.Factory.
type SupervisorFactory = supervisor.Factory

// NewSupervisor creates a new Supervisor for the given workers.
// Alias for pkg/supervisor.New.
func NewSupervisor(name string, strategy SupervisorStrategy, specs ...SupervisorSpec) Supervisor {
	return supervisor.New(name, strategy, specs...)
}
