package lifecycle

import (
	"context"
	"io"
	"iter"
	"os"
	"os/exec"
	"time"

	"log/slog"

	"github.com/aretw0/lifecycle/pkg/container"
	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/handlers"
	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
	"github.com/aretw0/lifecycle/pkg/proc"
	"github.com/aretw0/lifecycle/pkg/runtime"
	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/sources"
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

// Sleep pauses the current goroutine for at least the duration d.
// Alias for pkg/runtime.Sleep.
func Sleep(ctx context.Context, d time.Duration) error {
	return runtime.Sleep(ctx, d)
}

// Runnable defines a long-running process.
// Alias for pkg/runtime.Runnable.
type Runnable = runtime.Runnable

// Job creates a Runnable from a function.
// Alias for pkg/runtime.Job.
func Job(fn func(context.Context) error) Runnable {
	return runtime.Job(fn)
}

// Run executes the application logic with a managed SignalContext.
// Alias for pkg/runtime.Run.
func Run(r Runnable, opts ...SignalOption) error {
	return runtime.Run(r, opts...)
}

// OnShutdown safely registers a shutdown hook on the context if it supports it.
// It abstracts the type assertion for *signal.Context.
func OnShutdown(ctx context.Context, fn func()) {
	if sc, ok := ctx.(*signal.Context); ok {
		sc.OnShutdown(fn)
	}
	// If context is not a SignalContext, we could log a warning,
	// but purely functional options often fail silently or use an interface.
	// For now, silent allow is arguably least intrusive, but explicit is better.
	// We'll stick to simple casting for now.
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

// Do executes a function in a "Safe Executor" (Panic Recovery + Observability).
// Alias for pkg/runtime.Do.
func Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return runtime.Do(ctx, fn)
}

// DoDetached executes a function in a "Critical Section" (Detached Context).
// Alias for pkg/runtime.DoDetached.
func DoDetached(parent context.Context, fn func(ctx context.Context) error) error {
	return runtime.DoDetached(parent, fn)
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
// Alias for pkg/signal.Option.
type SignalOption = signal.Option

// SignalState represents the configuration state of the SignalContext.
// Alias for pkg/signal.State.
type SignalState = signal.State

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
	// WorkerEnvResumeID is the unique session identifier for a worker.
	WorkerEnvResumeID = worker.EnvResumeID
	// WorkerEnvPrevExit is the exit code of the previous execution of this worker.
	WorkerEnvPrevExit = worker.EnvPrevExit
)

// Supervisor defines the interface for a supervisor.
// Alias for pkg/supervisor.Supervisor.
type Supervisor = supervisor.Supervisor

// SupervisorStrategy defines how the supervisor handles child failures.
// Alias for pkg/supervisor.Strategy.
type SupervisorStrategy = supervisor.Strategy

const (
	// SupervisorStrategyOneForOne: If a child process terminates, only that process is restarted.
	SupervisorStrategyOneForOne = supervisor.StrategyOneForOne
	// SupervisorStrategyOneForAll: If a child process terminates, all other child processes are terminated.
	SupervisorStrategyOneForAll = supervisor.StrategyOneForAll
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

// --- Control Plane Aliases (v2.0) ---

// Event is a stimulus that triggers a reaction.
// Alias for pkg/control.Event.
type Event = control.Event

// Handler responds to an event.
// Alias for pkg/control.Handler.
type Handler = control.Handler

// HandlerFunc matches the signature of a Handler.
// Alias for pkg/control.HandlerFunc.
type HandlerFunc = control.HandlerFunc

// Source is a producer of events.
// Alias for pkg/control.Source.
type Source = control.Source

// Router maps events to reactions.
// Alias for pkg/control.Router.
type Router = control.Router

// RouterOption is an option for configuring a Router.
// Alias for pkg/control.RouterOption.
type RouterOption = control.RouterOption

// NewRouter creates a new Control Router.
// Alias for pkg/control.NewRouter.
func NewRouter(opts ...RouterOption) *Router {
	return control.NewRouter(opts...)
}

// Go starts a tracked goroutine.
// Alias for pkg/runtime.Go.
func Go(ctx context.Context, fn func(context.Context) error) {
	runtime.Go(ctx, fn)
}

// Receive creates a push iterator that yields values from the channel until
// the context is cancelled or the channel is closed.
// Alias for pkg/runtime.Receive.
func Receive[V any](ctx context.Context, ch <-chan V) iter.Seq[V] {
	return runtime.Receive(ctx, ch)
}

// NewOSSignalSource creates a source that listens for OS signals.
// Alias for pkg/sources.NewOSSignalSource.
func NewOSSignalSource(signals ...os.Signal) Source {
	return sources.NewOSSignalSource(signals...)
}

// NewWebhookSource creates a source that listens for Webhooks.
// Alias for pkg/sources.NewWebhookSource.
func NewWebhookSource(addr string) *sources.WebhookSource {
	return sources.NewWebhookSource(addr)
}

// NewShutdownHandler returns a handler that cancels context.
// Alias for pkg/handlers.NewShutdown.
func NewShutdownHandler(cancel context.CancelFunc) Handler {
	return handlers.NewShutdown(cancel)
}

// NewReloadHandler returns a handler that reloads configuration.
// Alias for pkg/handlers.NewReload.
func NewReloadHandler(onReload func(context.Context) error) Handler {
	return handlers.NewReload(onReload)
}

// TickEvent represents a periodic time tick.
// Alias for pkg/sources.TickEvent.
type TickEvent = sources.TickEvent

// NewTickerSource creates a source that emits periodic events.
// Alias for pkg/sources.NewTickerSource.
func NewTickerSource(interval time.Duration) Source {
	return sources.NewTickerSource(interval)
}

// SuspendHandler manages Suspend and Resume events.
// Alias for pkg/handlers.SuspendHandler.
type SuspendHandler = handlers.SuspendHandler

// SuspendEvent is the event that triggers a suspension.
// Alias for pkg/control.SuspendEvent.
type SuspendEvent = control.SuspendEvent

// ResumeEvent is the event that triggers a resumption.
// Alias for pkg/control.ResumeEvent.
type ResumeEvent = control.ResumeEvent

// NewSuspendHandler creates a new handler for suspend/resume events.
// Alias for pkg/handlers.NewSuspendHandler.
func NewSuspendHandler() *SuspendHandler {
	return handlers.NewSuspendHandler()
}

// HealthCheckSource runs a periodic health check.
// Alias for pkg/sources.HealthCheckSource.
type HealthCheckSource = sources.HealthCheckSource

// NewHealthCheckSource creates a new health monitor.
// Alias for pkg/sources.NewHealthCheckSource.
func NewHealthCheckSource(name string, check sources.CheckFunc, opts ...sources.HealthOption) *HealthCheckSource {
	return sources.NewHealthCheckSource(name, check, opts...)
}

// FileWatchSource watches for file changes via polling.
// Alias for pkg/sources.FileWatchSource.
type FileWatchSource = sources.FileWatchSource

// NewFileWatchSource creates a new source that polls the given file path.
// Alias for pkg/sources.NewFileWatchSource.
func NewFileWatchSource(path string, interval time.Duration) *FileWatchSource {
	return sources.NewFileWatchSource(path, interval)
}

// --- Stdlib Pattern Helpers ---

// Handle registers a handler on the DefaultRouter.
// Alias for pkg/control.Handle.
func Handle(pattern string, handler Handler) {
	control.Handle(pattern, handler)
}

// HandleFunc registers a handler function on the DefaultRouter.
// Alias for pkg/control.HandleFunc.
func HandleFunc(pattern string, handler func(context.Context, Event) error) {
	control.HandleFunc(pattern, handler)
}

// DefaultRouter is the default instance for package-level helpers.
// Alias for pkg/control.DefaultRouter.
var DefaultRouter = control.DefaultRouter
