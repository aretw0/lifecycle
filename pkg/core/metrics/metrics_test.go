package metrics

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/log"
)

func TestNoOpProvider(t *testing.T) {
	p := &NoOpProvider{}
	// Should not panic or do anything
	p.IncSignalReceived("INT")
	p.IncProcessStarted()
	p.IncProcessFailed()
	p.IncTerminalUpgrade(true)
}

func TestLogProvider(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}
	lp.IncSignalReceived("SIGINT")

	if !bytes.Contains(buf.Bytes(), []byte("metric incremented")) {
		t.Errorf("Expected log to contain 'metric incremented', got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_signals_total")) {
		t.Errorf("Expected log to contain 'lifecycle_signals_total', got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("SIGINT")) {
		t.Errorf("Expected log to contain 'SIGINT', got: %s", buf.String())
	}
}

func TestGlobalProvider(t *testing.T) {
	defaultP := GetProvider()
	if _, ok := defaultP.(*NoOpProvider); !ok {
		t.Errorf("Expected default provider to be NoOpProvider, got %T", defaultP)
	}

	mock := &NoOpProvider{}
	SetProvider(mock)
	if GetProvider() != mock {
		t.Error("SetProvider failed to update global provider")
	}

	SetProvider(nil)
	if _, ok := GetProvider().(*NoOpProvider); !ok {
		t.Error("SetProvider(nil) should restore NoOpProvider")
	}
}

// Comprehensive tests for LogProvider methods
func TestLogProvider_Signal_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}
	lp.IncSignalReceived("SIGTERM")

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_signals_total")) {
		t.Errorf("Expected 'lifecycle_signals_total' in output, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("SIGTERM")) {
		t.Errorf("Expected 'SIGTERM' in output, got: %s", output)
	}
}

func TestLogProvider_Process_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncProcessStarted()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_processes_started_total")) {
		t.Error("IncProcessStarted should log metric")
	}

	buf.Reset()
	lp.IncProcessFailed()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_processes_failed_total")) {
		t.Error("IncProcessFailed should log metric")
	}
}

func TestLogProvider_Terminal_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncTerminalUpgrade(true)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_terminal_upgrades_total")) {
		t.Error("IncTerminalUpgrade should log metric")
	}

	buf.Reset()
	lp.IncTerminalUpgrade(false)
	if !bytes.Contains(buf.Bytes(), []byte("success")) {
		t.Error("IncTerminalUpgrade should log success parameter")
	}
}

func TestLogProvider_Hook_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncHookExecuted()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_hooks_executed_total")) {
		t.Error("IncHookExecuted should log metric")
	}

	buf.Reset()
	lp.IncHookPanicked()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_hooks_panicked_total")) {
		t.Error("IncHookPanicked should log metric")
	}

	buf.Reset()
	lp.ObserveHookDuration(100 * time.Millisecond)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_hook_duration_seconds")) {
		t.Error("ObserveHookDuration should log metric")
	}
}

func TestLogProvider_Worker_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncWorkerStarted("background")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_workers_started_total")) {
		t.Error("IncWorkerStarted should log metric")
	}

	buf.Reset()
	lp.IncWorkerStopped("background")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_workers_stopped_total")) {
		t.Error("IncWorkerStopped should log metric")
	}

	buf.Reset()
	lp.IncWorkerFailed("background")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_workers_failed_total")) {
		t.Error("IncWorkerFailed should log metric")
	}

	buf.Reset()
	lp.ObserveWorkerDuration("background", 50*time.Millisecond)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_worker_duration_seconds")) {
		t.Error("ObserveWorkerDuration should log metric")
	}
}

func TestLogProvider_Supervisor_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncSupervisorRestart("api-pool", "OnFailure")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_supervisor_restarts_total")) {
		t.Error("IncSupervisorRestart should log metric")
	}

	buf.Reset()
	lp.IncChildRestart("api-pool", "worker-1")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_worker_restarts_total")) {
		t.Error("IncChildRestart should log metric")
	}

	buf.Reset()
	lp.IncSupervisorAdd("api-pool")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_supervisor_add_total")) {
		t.Error("IncSupervisorAdd should log metric")
	}

	buf.Reset()
	lp.IncSupervisorRemove("api-pool")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_supervisor_remove_total")) {
		t.Error("IncSupervisorRemove should log metric")
	}
}

func TestLogProvider_Backoff_And_Shutdown(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncBackoffTriggered("worker-1", 5*time.Second)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_backoff_triggered_seconds")) {
		t.Error("IncBackoffTriggered should log metric")
	}

	buf.Reset()
	lp.ObserveShutdownDuration("api", 2*time.Second)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_worker_shutdown_duration_seconds")) {
		t.Error("ObserveShutdownDuration should log metric")
	}
}

func TestLogProvider_Force_Exit_And_CircuitBreaker(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncForceExitTriggered()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_force_exit_triggered_total")) {
		t.Error("IncForceExitTriggered should log metric")
	}

	buf.Reset()
	lp.IncCircuitBreakerTriggered("worker-1")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_supervisor_circuit_breaker_triggered_total")) {
		t.Error("IncCircuitBreakerTriggered should log metric")
	}
}

func TestLogProvider_CriticalSection_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncCriticalSectionStarted()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_critical_section_started_total")) {
		t.Error("IncCriticalSectionStarted should log metric")
	}

	buf.Reset()
	lp.IncCriticalSectionFinished(true)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_critical_section_finished_total")) {
		t.Error("IncCriticalSectionFinished should log metric")
	}

	buf.Reset()
	lp.ObserveCriticalSectionDuration(100 * time.Millisecond)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_critical_section_duration_seconds")) {
		t.Error("ObserveCriticalSectionDuration should log metric")
	}
}

func TestLogProvider_Container_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncContainerStarted("nginx:latest")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_container_started_total")) {
		t.Error("IncContainerStarted should log metric")
	}

	buf.Reset()
	lp.IncContainerStopped("nginx:latest")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_container_stopped_total")) {
		t.Error("IncContainerStopped should log metric")
	}

	buf.Reset()
	lp.IncContainerFailed("nginx:latest")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_container_failed_total")) {
		t.Error("IncContainerFailed should log metric")
	}

	buf.Reset()
	lp.ObserveContainerDuration("nginx:latest", 3*time.Second)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_container_duration_seconds")) {
		t.Error("ObserveContainerDuration should log metric")
	}
}

func TestLogProvider_Goroutine_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncGoroutineStarted()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_goroutines_started_total")) {
		t.Error("IncGoroutineStarted should log metric")
	}

	buf.Reset()
	lp.IncGoroutineFinished()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_goroutines_finished_total")) {
		t.Error("IncGoroutineFinished should log metric")
	}

	buf.Reset()
	lp.IncGoroutinePanicked()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_goroutines_panicked_total")) {
		t.Error("IncGoroutinePanicked should log metric")
	}

	buf.Reset()
	lp.ObserveGoroutineBlockDuration(10 * time.Millisecond)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_goroutines_block_duration_seconds")) {
		t.Error("ObserveGoroutineBlockDuration should log metric")
	}

	buf.Reset()
	lp.IncGoroutineWaiting()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_goroutines_waiting_current")) {
		t.Error("IncGoroutineWaiting should log metric")
	}

	buf.Reset()
	lp.DecGoroutineWaiting()
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_goroutines_waiting_current")) {
		t.Error("DecGoroutineWaiting should log metric")
	}
}

func TestLogProvider_Event_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.SetLogger(logger)

	lp := &LogProvider{}

	buf.Reset()
	lp.IncEventEmitted("OSSignal")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_events_emitted_total")) {
		t.Error("IncEventEmitted should log metric")
	}

	buf.Reset()
	lp.IncEventRouted("lifecycle.shutdown")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_events_routed_total")) {
		t.Error("IncEventRouted should log metric")
	}

	buf.Reset()
	lp.IncHandlerExecuted("lifecycle.shutdown")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_handlers_executed_total")) {
		t.Error("IncHandlerExecuted should log metric")
	}

	buf.Reset()
	lp.IncHandlerError("lifecycle.shutdown", errors.New("test error"))
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_handler_errors_total")) {
		t.Error("IncHandlerError should log metric")
	}

	buf.Reset()
	lp.ObserveHandlerDuration("lifecycle.shutdown", 50*time.Millisecond)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_handler_duration_seconds")) {
		t.Error("ObserveHandlerDuration should log metric")
	}

	buf.Reset()
	lp.ObserveEventBlockDuration("OSSignal", 100*time.Millisecond)
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_events_block_duration_seconds")) {
		t.Error("ObserveEventBlockDuration should log metric")
	}

	buf.Reset()
	lp.IncEventWaiting("OSSignal")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_events_waiting_current")) {
		t.Error("IncEventWaiting should log metric")
	}

	buf.Reset()
	lp.DecEventWaiting("OSSignal")
	if !bytes.Contains(buf.Bytes(), []byte("lifecycle_events_waiting_current")) {
		t.Error("DecEventWaiting should log metric")
	}
}

func TestNoOpProvider_All_Methods(t *testing.T) {
	p := &NoOpProvider{}
	// All these should not panic
	p.IncSignalReceived("SIGINT")
	p.IncProcessStarted()
	p.IncProcessFailed()
	p.IncTerminalUpgrade(true)
	p.IncHookExecuted()
	p.IncHookPanicked()
	p.ObserveHookDuration(100 * time.Millisecond)
	p.IncWorkerStarted("test")
	p.IncWorkerStopped("test")
	p.IncWorkerFailed("test")
	p.ObserveWorkerDuration("test", 100*time.Millisecond)
	p.IncSupervisorRestart("sup", "OnFailure")
	p.IncChildRestart("sup", "child")
	p.IncSupervisorAdd("sup")
	p.IncSupervisorRemove("sup")
	p.IncBackoffTriggered("child", 5*time.Second)
	p.ObserveShutdownDuration("test", 2*time.Second)
	p.IncForceExitTriggered()
	p.IncCircuitBreakerTriggered("child")
	p.IncCriticalSectionStarted()
	p.IncCriticalSectionFinished(true)
	p.ObserveCriticalSectionDuration(100 * time.Millisecond)
	p.IncContainerStarted("image")
	p.IncContainerStopped("image")
	p.IncContainerFailed("image")
	p.ObserveContainerDuration("image", 3*time.Second)
	p.IncGoroutineStarted()
	p.IncGoroutineFinished()
	p.IncGoroutinePanicked()
	p.ObserveGoroutineBlockDuration(10 * time.Millisecond)
	p.IncGoroutineWaiting()
	p.DecGoroutineWaiting()
	p.IncEventEmitted("OSSignal")
	p.IncEventRouted("lifecycle.shutdown")
	p.IncHandlerExecuted("lifecycle.shutdown")
	p.IncHandlerError("lifecycle.shutdown", nil)
	p.ObserveHandlerDuration("lifecycle.shutdown", 50*time.Millisecond)
	p.ObserveEventBlockDuration("OSSignal", 100*time.Millisecond)
	p.IncEventWaiting("OSSignal")
	p.DecEventWaiting("OSSignal")
}



