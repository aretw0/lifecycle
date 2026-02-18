//go:build stress

package worker_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/worker"
)

func TestStress_ProcessChurn(t *testing.T) {
	// This test churns processes rapidly to ensure no leaks or race conditions in start/stop logic.
	// It uses the same HelperProcess pattern as process_test.go.

	os.Setenv(HelperProcess, "1")
	defer os.Unsetenv(HelperProcess)

	ctx := context.Background()
	iterations := 50

	t.Logf("Starting stress test with %d iterations...", iterations)

	for i := 0; i < iterations; i++ {
		// Use "sleep" mode which waits until stopped
		w := worker.NewProcessWorker("stress-worker", os.Args[0], "sleep")

		start := time.Now()
		if err := w.Start(ctx); err != nil {
			t.Fatalf("Iteration %d: Start failed: %v", i, err)
		}

		// Let it run for a tiny bit to ensure it actually initializes
		time.Sleep(10 * time.Millisecond)

		// Stop it
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		err := w.Stop(stopCtx)
		cancel()

		if err != nil {
			if err == context.DeadlineExceeded {
				// No Windows, é esperado que processos não respondam a sinais.
				// No Linux, isso é considerado falha.
				if isWindows() {
					t.Logf("[Windows] Iteration %d: Stop timeout (context.DeadlineExceeded) - comportamento esperado", i)
				} else {
					t.Fatalf("[Linux] Iteration %d: Stop failed: %v", i, err)
				}
			} else {
				t.Fatalf("Iteration %d: Stop failed: %v", i, err)
			}
		}

		// Ensure Wait closes
		select {
		case <-w.Wait():
			// OK
		case <-time.After(1 * time.Second):
			t.Fatalf("Iteration %d: Wait timed out", i)
		}

		if i%10 == 0 {
			t.Logf("Completed %d iterations. Duration: %v", i, time.Since(start))
		}
	}

	t.Log("Stress test completed successfully.")
}

// isWindows retorna true se o teste está rodando no Windows.
func isWindows() bool {
	return os.PathSeparator == '\\'
}
