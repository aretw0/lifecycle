package lifecycle

import (
	"context"
	"io"
	"iter"
	"os"
	"os/exec"
	"time"

	"log/slog"

	"github.com/aretw0/lifecycle/pkg/core/container"
	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/proc"
	"github.com/aretw0/lifecycle/pkg/core/runtime"
	"github.com/aretw0/lifecycle/pkg/core/signal"
	"github.com/aretw0/lifecycle/pkg/core/supervisor"
	"github.com/aretw0/lifecycle/pkg/core/termio"
	"github.com/aretw0/lifecycle/pkg/core/worker"
	"github.com/aretw0/lifecycle/pkg/core/worker/suspend"
	"github.com/aretw0/lifecycle/pkg/events"
)

// ======================================================================================
// 1. Core Runtime
// ======================================================================================

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
func Run(r Runnable, opts ...any) error {
	return runtime.Run(r, opts...)
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

// ======================================================================================
// 2. Signal Context
// ======================================================================================

// Context represents the signal context.
type Context = signal.Context

// Option is a functional option for signal configuration.
// Alias for pkg/signal.Option.
type SignalOption = signal.Option

// SignalState returns a snapshot of the current configuration.
// Alias for signal.State.
type SignalState = signal.State

// NewSignalContext creates a context that cancels on SIGTERM/SIGINT.
// On the first signal, context is cancelled. On the second, it force exits.
// Behavior can be customized via functional options.
// Alias for pkg/signal.NewContext.
func NewSignalContext(parent context.Context, opts ...signal.Option) *signal.Context {
	return signal.NewContext(parent, opts...)
}

// WithForceExit configures the threshold of signals required to trigger an immediate os.Exit(1).
// Threshold values:
// 1 (Default): SIGINT cancels context immediately.
// n >= 2: SIGINT is captured, os.Exit(1) at n-th signal (Escalation Mode).
// 0 (Unsafe): Automatic os.Exit(1) is disabled for SIGINT.
func WithForceExit(threshold int) signal.Option {
	return signal.WithForceExit(threshold)
}

// WithResetTimeout configures the duration after which the signal count resets.
// Alias for pkg/signal.WithResetTimeout.
func WithResetTimeout(d time.Duration) signal.Option {
	return signal.WithResetTimeout(d)
}

// WithHookTimeout configures the duration after which a running hook produces a warning log.
// Alias for pkg/signal.WithHookTimeout.
func WithHookTimeout(d time.Duration) signal.Option {
	return signal.WithHookTimeout(d)
}

// WithCancelOnInterrupt controls whether SIGINT automatically cancels the context.
// Alias for pkg/signal.WithCancelOnInterrupt.
func WithCancelOnInterrupt(enabled bool) signal.Option {
	return signal.WithCancelOnInterrupt(enabled)
}

// OnShutdown safely registers a shutdown hook on the context if it supports it.
// It abstracts the discovery of signal.Context even when wrapped.
func OnShutdown(ctx context.Context, fn func()) {
	if sc, ok := signal.FromContext(ctx); ok {
		sc.OnShutdown(fn)
	}
}

// Shutdown initiates a graceful shutdown of the application asynchronously.
// It cancels the context and triggers all registered OnShutdown hooks.
//
// When to use:
//   - Inside an HTTP handler or RPC method where blocking is undesirable.
//   - When you want to trigger shutdown but don't need to wait for it (fire-and-forget).
//   - In tests where you want to assert intermediate states during shutdown.
//
// Difference from Stop():
//   - Shutdown() triggers application teardown (hooks run, context cancels).
//   - Stop() only stops signal monitoring (hooks do NOT run, context remains valid).
func Shutdown(ctx context.Context) {
	if sc, ok := signal.FromContext(ctx); ok {
		sc.Shutdown()
	}
}

// ShutdownAndWait initiates a graceful shutdown and blocks until all hooks have completed.
// It is a shorthand for Shutdown(ctx) followed by Wait(ctx).
//
// When to use:
//   - In your main() function to ensure strict cleanup before exit.
//   - When the next line of code assumes all resources are released.
func ShutdownAndWait(ctx context.Context) {
	if sc, ok := signal.FromContext(ctx); ok {
		sc.ShutdownWait()
	}
}

// Wait blocks until all shutdown hooks have finished.
func Wait(ctx context.Context) {
	if sc, ok := signal.FromContext(ctx); ok {
		sc.Wait()
	}
}

// IsUnsafe returns true if the context is configured to never force exit.
func IsUnsafe(ctx context.Context) bool {
	if sc, ok := signal.FromContext(ctx); ok {
		return sc.IsUnsafe()
	}
	return false
}

// GetForceExitThreshold returns the number of signals required to trigger os.Exit(1).
func GetForceExitThreshold(ctx context.Context) int {
	if sc, ok := signal.FromContext(ctx); ok {
		return sc.ForceExitThreshold()
	}
	return 0
}

// GetSignalState returns a snapshot of the context's signal configuration.
func GetSignalState(ctx context.Context) (SignalState, bool) {
	if sc, ok := signal.FromContext(ctx); ok {
		return sc.State(), true
	}
	return SignalState{}, false
}

// Signal returns the signal that caused the context to be cancelled/interrupted, or nil.
// It safely unwraps the context to find the SignalContext.
func Signal(ctx context.Context) os.Signal {
	if sc, ok := signal.FromContext(ctx); ok {
		return sc.Signal()
	}
	return nil
}

// SignalStateDiagram returns a Mermaid state diagram string representing the signal context configuration.
// Alias for pkg/signal.MermaidState.
func SignalStateDiagram(s SignalState) string {
	return signal.MermaidState(s)
}

// ======================================================================================
// 3. Workers & Supervisor
// ======================================================================================

// Worker defines the interface for a managed unit of work.
// Alias for pkg/worker.Worker.
type Worker = worker.Worker

// Suspendable defines a worker that can pause its execution in-place without exiting.
// Alias for pkg/worker.Suspendable.
type Suspendable = worker.Suspendable

// WorkerState represents the snapshot of a worker's state.
type WorkerState = worker.State

// WorkerStatus represents the lifecycle state of a worker.
type WorkerStatus = worker.Status

const (
	WorkerStatusCreated = worker.StatusCreated
	WorkerStatusPending = worker.StatusPending
	WorkerStatusRunning = worker.StatusRunning
	WorkerStatusStopped = worker.StatusStopped
	WorkerStatusFailed  = worker.StatusFailed
)

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

// BaseWorker provides default implementations for Worker interface boilerplate.
// Embed this in your worker types to avoid repeating Stop/Wait/String methods.
//
// Example:
//
//	type MyWorker struct {
//	    *lifecycle.BaseWorker
//	    // custom fields...
//	}
//
//	func (w *MyWorker) State() lifecycle.WorkerState {
//	    return w.ExportState(func(s *lifecycle.WorkerState) {
//	        s.Metadata = map[string]string{"custom": "data"}
//	    })
//	}
//
//	func NewMyWorker() *MyWorker {
//	    return &MyWorker{
//	        BaseWorker: lifecycle.NewBaseWorker("MyWorker"),
//	    }
//	}
//
//	func (w *MyWorker) Start(ctx context.Context) error {
//	    return w.StartFunc(ctx, w.Run)
//	}
//
// Alias for pkg/worker.BaseWorker.
type BaseWorker = worker.BaseWorker

// NewBaseWorker creates a BaseWorker with the given name.
// The name is immutable (construct a new worker to change it).
// Alias for pkg/worker.NewBaseWorker.
func NewBaseWorker(name string) *BaseWorker {
	return worker.NewBaseWorker(name)
}

// SuspendManager provides channel-based suspend/resume with context cancellation.
// Use this for workers that need to support cancellable suspension.
// For maximum performance (>10k suspends/sec), use sync.Cond directly.
//
// Example:
//
//	type MyWorker struct {
//	    lifecycle.BaseWorker
//	    suspendMgr *lifecycle.SuspendManager
//	}
//
//	func (w *MyWorker) Run(ctx context.Context) error {
//	    for {
//	        if err := w.suspendMgr.Wait(ctx); err != nil {
//	            return err
//	        }
//	        // Do work...
//	    }
//	}
//
// See examples/suspend/channels/ for complete example.
// Alias for pkg/worker/suspend.Manager.
type SuspendManager = suspend.Manager

// NewSuspendManager creates a suspend manager (initially running).
// Alias for pkg/worker/suspend.NewManager.
func NewSuspendManager() *SuspendManager {
	return suspend.NewManager()
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

// SupervisorRestartPolicy defines when a child worker should be restarted.
// Alias for pkg/supervisor.RestartPolicy.
type SupervisorRestartPolicy = supervisor.RestartPolicy

const (
	RestartAlways    = supervisor.RestartAlways
	RestartOnFailure = supervisor.RestartOnFailure
	RestartNever     = supervisor.RestartNever
)

// SupervisorFactory is a function that creates a new worker instance.
// Alias for pkg/supervisor.Factory.
type SupervisorFactory = supervisor.Factory

// NewSupervisor creates a new Supervisor for the given workers.
// Alias for pkg/supervisor.New.
func NewSupervisor(name string, strategy SupervisorStrategy, specs ...SupervisorSpec) Supervisor {
	return supervisor.New(name, strategy, specs...)
}

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

// ======================================================================================
// 4. Control Plane (v2)
// ======================================================================================

// Event is a stimulus that triggers a reaction.
// Alias for pkg/events.Event.
type Event = events.Event

// Handler responds to an event.
// Alias for pkg/events.Handler.
type Handler = events.Handler

// HandlerFunc matches the signature of a Handler.
// Alias for pkg/events.HandlerFunc.
type HandlerFunc = events.HandlerFunc

// Source is a producer of events.
// Alias for pkg/events.Source.
type Source = events.Source

// Router maps events to reactions.
// Alias for pkg/events.events.
type Router = events.Router

// RouterOption is an option for configuring a events.
// Alias for pkg/events.RouterOption.
type RouterOption = events.RouterOption

// NewRouter creates a new Control events.
// Alias for pkg/events.Newevents.
func NewRouter(opts ...RouterOption) *Router {
	return events.NewRouter(opts...)
}

// DefaultRouter is the default instance for package-level helpers.
// Alias for pkg/events.Defaultevents.
var DefaultRouter = events.DefaultRouter

// Handle registers a handler on the Defaultevents.
// Alias for pkg/events.Handle.
func Handle(pattern string, handler Handler) {
	events.Handle(pattern, handler)
}

// HandleFunc registers a handler function on the Defaultevents.
// Alias for pkg/events.HandleFunc.
func HandleFunc(pattern string, handler func(context.Context, Event) error) {
	events.HandleFunc(pattern, handler)
}

// TerminateEvent is a high-level event that chains Suspend and Shutdown.
// Alias for pkg/events.TerminateEvent.
type TerminateEvent = events.TerminateEvent

// TerminateOption configures the TerminateHandler.
// Alias for pkg/events.TerminateOption.
type TerminateOption = events.TerminateOption

// WithContinueOnFailure configures whether to proceed with shutdown even if suspension fails.
// Alias for pkg/events.WithContinueOnFailure.
func WithContinueOnFailure(continueOnFailure bool) TerminateOption {
	return events.WithContinueOnFailure(continueOnFailure)
}

// NewTerminateHandler creates a new handler that chains suspension and shutdown.
// Alias for pkg/events.NewTerminate.
func NewTerminateHandler(suspend Handler, shutdown Handler, opts ...TerminateOption) Handler {
	return events.NewTerminate(suspend, shutdown, opts...)
}

// ======================================================================================
// 5. Sources (Event Producers)
// ======================================================================================

// NewOSSignalSource creates a source that listens for OS signals.
// Alias for pkg/events.NewOSSignalSource.
func NewOSSignalSource(signals ...os.Signal) Source {
	return events.NewOSSignalSource(signals...)
}

// InputSource reads from Stdin and emits events.
// Alias for pkg/events.InputSource.
type InputSource = events.InputSource

// InputOption configures the InputSource.
// Alias for pkg/events.InputOption.
type InputOption = events.InputOption

// NewInputSource creates a new source that listens for standard CLI commands.
// Alias for pkg/events.NewInputSource.
func NewInputSource(opts ...InputOption) *InputSource {
	return events.NewInputSource(opts...)
}

// WithUnknownHandler configures a custom handler for unknown commands.
// Alias for pkg/events.WithUnknownHandler.
func WithUnknownHandler(fn func(cmd string, known []string)) InputOption {
	return events.WithUnknownHandler(fn)
}

// WithInputMapping adds a custom command mapping.
// Alias for pkg/events.WithInputMapping.
func WithInputMapping(key string, event events.Event) InputOption {
	return events.WithInputMapping(key, event)
}

// WithInputBackoff configures the duration to wait before retrying interruptions or errors.
// Alias for pkg/events.WithInputBackoff.
func WithInputBackoff(d time.Duration) InputOption {
	return events.WithInputBackoff(d)
}

// InputEvent represents a generic text command.
// Alias for pkg/events.InputEvent.
type InputEvent = events.InputEvent

// TickEvent represents a periodic time tick.
// Alias for pkg/events.TickEvent.
type TickEvent = events.TickEvent

// NewTickerSource creates a source that emits periodic events.
// Alias for pkg/events.NewTickerSource.
func NewTickerSource(interval time.Duration) Source {
	return events.NewTickerSource(interval)
}

// HealthCheckSource runs a periodic health check.
// Alias for pkg/events.HealthCheckSource.
type HealthCheckSource = events.HealthCheckSource

// HealthOption configures a HealthCheckSource.
// Alias for pkg/events.HealthOption.
type HealthOption = events.HealthOption

// NewHealthCheckSource creates a new health monitor.
// Alias for pkg/events.NewHealthCheckSource.
func NewHealthCheckSource(name string, check events.CheckFunc, opts ...HealthOption) *HealthCheckSource {
	return events.NewHealthCheckSource(name, check, opts...)
}

// WithHealthInterval sets the check interval.
// Alias for pkg/events.WithHealthInterval.
func WithHealthInterval(d time.Duration) HealthOption {
	return events.WithHealthInterval(d)
}

// WithHealthStrategy sets the triggering strategy (Edge vs Level).
// Alias for pkg/events.WithHealthStrategy.
func WithHealthStrategy(strategy events.TriggerStrategy) HealthOption {
	return events.WithHealthStrategy(strategy)
}

// FileWatchSource watches for file changes using platform-specific notifications.
// Alias for pkg/events.FileWatchSource.
type FileWatchSource = events.FileWatchSource

// NewFileWatchSource creates a new source that watches the given file for changes.
// Uses fsnotify for efficient, event-driven file watching (Linux, Windows, macOS, BSD).
//
// Example:
//
//	events.AddSource(lifecycle.NewFileWatchSource("config.yaml"))
//	events.Handle("file/*", lifecycle.NewReloadHandler(reloadConfig))
//
// Alias for pkg/events.NewFileWatchSource.
func NewFileWatchSource(path string) *FileWatchSource {
	return events.NewFileWatchSource(path)
}

// NewWebhookSource creates a source that listens for Webhooks.
// Alias for pkg/events.NewWebhookSource.
func NewWebhookSource(addr string) *events.WebhookSource {
	return events.NewWebhookSource(addr)
}

// ChannelSource adapts a generic Go channel to the Source interface.
// Alias for pkg/events.ChannelSource.
type ChannelSource = events.ChannelSource

// NewChannelSource creates a new source that reads from the given channel.
// Alias for pkg/events.NewChannelSource.
func NewChannelSource(ch <-chan Event) *ChannelSource {
	return events.NewChannelSource(ch)
}

// ======================================================================================
// 6. Handlers (Event Reactions)
// ======================================================================================

// ShutdownEvent is triggered when the application should shut down gracefully.
// Alias for pkg/events.ShutdownEvent.
type ShutdownEvent = events.ShutdownEvent

// NewShutdownHandler returns a handler that cancels context.
// It is automatically wrapped in events.Once to ensure idempotency.
// Alias for pkg/events.NewShutdown.
func NewShutdownHandler(cancel context.CancelFunc) Handler {
	return events.NewShutdown(cancel)
}

// NewShutdownFunc returns a handler that executes the given function once.
// Useful for wrapping generic close/cleanup operations as shutdown triggers.
// Alias for pkg/events.NewShutdownFunc.
func NewShutdownFunc(fn func()) Handler {
	return events.NewShutdownFunc(fn)
}

// NewReloadHandler returns a handler that reloads configuration.
// Alias for pkg/events.NewReload.
func NewReloadHandler(onReload func(context.Context) error) Handler {
	return events.NewReload(onReload)
}

// SuspendHandler manages Suspend and Resume events.
// Alias for pkg/events.SuspendHandler.
type SuspendHandler = events.SuspendHandler

// SuspendEvent is the event that triggers a suspension.
// Alias for pkg/events.SuspendEvent.
type SuspendEvent = events.SuspendEvent

// ResumeEvent is the event that triggers a resumption.
// Alias for pkg/events.ResumeEvent.
type ResumeEvent = events.ResumeEvent

// NewSuspendHandler creates a new handler for suspend/resume events.
// Alias for pkg/events.NewSuspendHandler.
func NewSuspendHandler() *SuspendHandler {
	return events.NewSuspendHandler()
}

// SmartSignalHandler implements "Double-Tap" signal logic (Suspend -> Quit).
// Alias for pkg/events.SmartSignalHandler.
type SmartSignalHandler = events.SmartSignalHandler

// NewSmartSignalHandler creates a new smart signal handler.
// Alias for pkg/events.NewSmartSignalHandler.
func NewSmartSignalHandler(s *SuspendHandler, q Handler) *SmartSignalHandler {
	return events.NewSmartSignalHandler(s, q)
}

// ClearLineEvent is triggered when an interactive input is interrupted.
// Alias for pkg/events.ClearLineEvent.
type ClearLineEvent = events.ClearLineEvent

// ======================================================================================
// 6a. Interactive Router Helpers
// ======================================================================================

// See interactive.go for NewInteractiveRouter and related options.

// ======================================================================================
// 7. Process Management (Low Level)
// ======================================================================================

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

// ======================================================================================
// 8. Terminal & I/O
// ======================================================================================

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

// ======================================================================================
// 9. Observability & Metrics
// ======================================================================================

// SetLogger overrides the global logger used by the library.
// Alias for pkg/log.SetLogger.
func SetLogger(l *slog.Logger) {
	log.SetLogger(l)
}

// WithLogger returns a RunOption to configure the global logger.
// Alias for pkg/runtime.WithLogger.
func WithLogger(l *slog.Logger) any {
	return runtime.WithLogger(l)
}

// SetMetricsProvider overrides the global metrics provider.
// This allowing bridging library metrics to Prometheus, OTEL, etc.
// Alias for pkg/metrics.SetProvider.
func SetMetricsProvider(p metrics.Provider) {
	metrics.SetProvider(p)
}

// WithMetrics returns a RunOption to configure the global metrics provider.
// Alias for pkg/runtime.WithMetrics.
func WithMetrics(p metrics.Provider) any {
	return runtime.WithMetrics(p)
}

// WithShutdownTimeout returns a RunOption to configure the diagnostic timeout during shutdown.
// Alias for pkg/runtime.WithShutdownTimeout.
func WithShutdownTimeout(d time.Duration) any {
	return runtime.WithShutdownTimeout(d)
}

// NewLogMetricsProvider returns a metrics provider that logs to the current logger.
// Useful for development and local verification.
// Alias for pkg/metrics.LogProvider.
func NewLogMetricsProvider() metrics.Provider {
	return &metrics.LogProvider{}
}
