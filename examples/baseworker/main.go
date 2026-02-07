package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aretw0/lifecycle"
)

// SimpleWorker demonstrates minimal boilerplate using BaseWorker embedding.
type SimpleWorker struct {
	*lifecycle.BaseWorker
	message string
}

func NewSimpleWorker(message string) *SimpleWorker {
	return &SimpleWorker{
		BaseWorker: lifecycle.NewBaseWorker("SimpleWorker"),
		message:    message,
	}
}

func (w *SimpleWorker) Start(ctx context.Context) error {
	return w.StartFunc(ctx, w.Run)
}

func (w *SimpleWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("SimpleWorker stopping")
			return ctx.Err()
		case <-ticker.C:
			slog.Info("SimpleWorker tick", "message", w.message)
		}
	}
}

// ===================================================================
// Compare with old boilerplate approach (commented for reference)
// ===================================================================

// type OldStyleWorker struct {
//     message string
//     done    chan error  // ← Had to manage this manually
// }
//
// func (w *OldStyleWorker) Start(ctx context.Context) error {
//     w.done = make(chan error, 1)  // ← Boilerplate
//     go func() {                    // ← Boilerplate
//         w.done <- w.Run(ctx)       // ← Boilerplate
//         close(w.done)              // ← Boilerplate
//     }()                            // ← Boilerplate
//     return nil
// }
//
// func (w *OldStyleWorker) Stop(ctx context.Context) error { return nil }  // ← Boilerplate
// func (w *OldStyleWorker) Wait() <-chan error             { return w.done }  // ← Boilerplate
// func (w *OldStyleWorker) String() string                 { return "OldStyle" }  // ← Boilerplate
// func (w *OldStyleWorker) State() lifecycle.WorkerState {  // ← Boilerplate
//     return lifecycle.WorkerState{Name: "OldStyle"}  // ← Boilerplate
// }  // ← Boilerplate
//
// Total: ~15 lines of boilerplate vs 2 lines with BaseWorker!

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	fmt.Println("=== BaseWorker Example ===")
	fmt.Println("Demonstrating reduced boilerplate with embedding pattern")
	fmt.Println()

	worker := NewSimpleWorker("Hello from BaseWorker!")

	// Verify embedding gives us all methods
	fmt.Printf("Worker name: %s\n", worker.String())
	fmt.Printf("Worker state: %+v\n", worker.State())
	fmt.Println()

	// Run with supervisor
	sup := lifecycle.NewSupervisor("demo", lifecycle.SupervisorStrategyOneForOne,
		lifecycle.SupervisorSpec{
			Name: "simple",
			Factory: func() (lifecycle.Worker, error) {
				return NewSimpleWorker("Supervised worker!"), nil
			},
			RestartPolicy: lifecycle.RestartOnFailure,
		},
	)

	lifecycle.Run(sup)
	fmt.Println("\n✅ Clean shutdown")
}
