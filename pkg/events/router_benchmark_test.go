package events

// Benchmarks for event router performance characteristics.
//
// Measures exact vs glob matching, middleware overhead, throughput, and scaling.
// All benchmarks use synthetic events to isolate router performance.

import (
	"context"
	"fmt"
	"testing"
)

// benchEvent is a simple event type for benchmarking.
type benchEvent struct {
	topic string
}

func (e *benchEvent) String() string {
	return e.topic
}

// BenchmarkRouterExactMatch measures exact route lookup performance.
func BenchmarkRouterExactMatch(b *testing.B) {
	router := NewRouter()
	ctx := context.Background()

	// Register exact routes
	for i := 0; i < 100; i++ {
		pattern := fmt.Sprintf("exact/route/%d", i)
		router.HandleFunc(pattern, noopHandler)
	}

	// Benchmark exact match (map lookup - O(1))
	event := &benchEvent{topic: "exact/route/50"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Dispatch(ctx, event)
	}
}

// BenchmarkRouterGlobMatch10 measures glob pattern matching with 10 routes.
func BenchmarkRouterGlobMatch10(b *testing.B) {
	router := NewRouter()
	ctx := context.Background()

	// Register glob patterns
	for i := 0; i < 10; i++ {
		pattern := fmt.Sprintf("signal/interrupt/%d/*", i)
		router.HandleFunc(pattern, noopHandler)
	}

	// Benchmark glob match (linear search - O(n))
	event := &benchEvent{topic: "signal/interrupt/5/handler"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Dispatch(ctx, event)
	}
}

// BenchmarkRouterGlobMatch100 measures glob pattern matching with 100 routes.
func BenchmarkRouterGlobMatch100(b *testing.B) {
	router := NewRouter()
	ctx := context.Background()

	// Register glob patterns
	for i := 0; i < 100; i++ {
		pattern := fmt.Sprintf("signal/interrupt/%d/*", i)
		router.HandleFunc(pattern, noopHandler)
	}

	// Benchmark glob match (linear search - O(n))
	event := &benchEvent{topic: "signal/interrupt/50/handler"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Dispatch(ctx, event)
	}
}

// BenchmarkRouterGlobMatch1000 measures glob pattern matching with 1000 routes.
func BenchmarkRouterGlobMatch1000(b *testing.B) {
	router := NewRouter()
	ctx := context.Background()

	// Register glob patterns
	for i := 0; i < 1000; i++ {
		pattern := fmt.Sprintf("signal/interrupt/%d/*", i)
		router.HandleFunc(pattern, noopHandler)
	}

	// Benchmark glob match (linear search - O(n))
	event := &benchEvent{topic: "signal/interrupt/500/handler"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Dispatch(ctx, event)
	}
}

// BenchmarkRouterMixedPatterns measures performance with mixed exact and glob routes.
func BenchmarkRouterMixedPatterns(b *testing.B) {
	router := NewRouter()
	ctx := context.Background()

	// Register 50 exact + 50 glob patterns
	for i := 0; i < 50; i++ {
		router.HandleFunc(fmt.Sprintf("exact/%d", i), noopHandler)
		router.HandleFunc(fmt.Sprintf("glob/%d/*", i), noopHandler)
	}

	b.Run("ExactMatch", func(b *testing.B) {
		event := &benchEvent{topic: "exact/25"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})

	b.Run("GlobMatch", func(b *testing.B) {
		event := &benchEvent{topic: "glob/25/handler"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		event := &benchEvent{topic: "unknown/route"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})
}

// BenchmarkRouterMiddleware measures middleware chain overhead.
func BenchmarkRouterMiddleware(b *testing.B) {
	b.Run("NoMiddleware", func(b *testing.B) {
		router := NewRouter()
		router.HandleFunc("test", noopHandler)
		ctx := context.Background()
		event := &benchEvent{topic: "test"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})

	b.Run("1Middleware", func(b *testing.B) {
		router := NewRouter()
		router.Use(noopMiddleware)
		router.HandleFunc("test", noopHandler)
		ctx := context.Background()
		event := &benchEvent{topic: "test"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})

	b.Run("5Middleware", func(b *testing.B) {
		router := NewRouter()
		for i := 0; i < 5; i++ {
			router.Use(noopMiddleware)
		}
		router.HandleFunc("test", noopHandler)
		ctx := context.Background()
		event := &benchEvent{topic: "test"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})

	b.Run("10Middleware", func(b *testing.B) {
		router := NewRouter()
		for i := 0; i < 10; i++ {
			router.Use(noopMiddleware)
		}
		router.HandleFunc("test", noopHandler)
		ctx := context.Background()
		event := &benchEvent{topic: "test"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})
}

// BenchmarkRouterThroughput measures events/second with concurrent dispatching.
func BenchmarkRouterThroughput(b *testing.B) {
	router := NewRouter()
	for i := 0; i < 50; i++ {
		pattern := fmt.Sprintf("event/%d", i)
		router.HandleFunc(pattern, noopHandler)
	}

	ctx := context.Background()

	b.Run("Sequential", func(b *testing.B) {
		event := &benchEvent{topic: "event/25"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
	})

	b.Run("Parallel", func(b *testing.B) {
		event := &benchEvent{topic: "event/25"}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				router.Dispatch(ctx, event)
			}
		})
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
	})
}

// BenchmarkRouterRegistration measures route registration overhead.
func BenchmarkRouterRegistration(b *testing.B) {
	b.Run("RegisterExact", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router := NewRouter()
			for j := 0; j < 100; j++ {
				pattern := fmt.Sprintf("route/%d", j)
				router.HandleFunc(pattern, noopHandler)
			}
		}
	})

	b.Run("RegisterGlob", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router := NewRouter()
			for j := 0; j < 100; j++ {
				pattern := fmt.Sprintf("route/%d/*", j)
				router.HandleFunc(pattern, noopHandler)
			}
		}
	})
}

// BenchmarkRouterIntrospection measures State() and Routes() overhead.
func BenchmarkRouterIntrospection(b *testing.B) {
	router := NewRouter()
	for i := 0; i < 100; i++ {
		pattern := fmt.Sprintf("route/%d/*", i)
		router.HandleFunc(pattern, noopHandler)
	}

	b.Run("State", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = router.State()
		}
	})

	b.Run("Routes", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = router.Routes()
		}
	})
}

// BenchmarkRouterConcurrentModification measures read/write contention.
func BenchmarkRouterConcurrentModification(b *testing.B) {
	router := NewRouter()
	ctx := context.Background()
	event := &benchEvent{topic: "test/route"}

	// Pre-populate with some routes
	for i := 0; i < 50; i++ {
		pattern := fmt.Sprintf("test/%d", i)
		router.HandleFunc(pattern, noopHandler)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				// 10% writes (registration)
				pattern := fmt.Sprintf("dynamic/%d", i)
				router.HandleFunc(pattern, noopHandler)
			} else {
				// 90% reads (dispatch)
				router.Dispatch(ctx, event)
			}
			i++
		}
	})
}

// BenchmarkRouterMemory measures memory allocation patterns.
func BenchmarkRouterMemory(b *testing.B) {
	b.Run("DispatchAllocation", func(b *testing.B) {
		router := NewRouter()
		router.HandleFunc("test", noopHandler)
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event := &benchEvent{topic: "test"}
			router.Dispatch(ctx, event)
		}
	})

	b.Run("RouterCreation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router := NewRouter()
			for j := 0; j < 10; j++ {
				pattern := fmt.Sprintf("route/%d", j)
				router.HandleFunc(pattern, noopHandler)
			}
		}
	})
}

// Helper functions

func noopHandler(ctx context.Context, e Event) error {
	return nil
}

func noopMiddleware(next Handler) Handler {
	return HandlerFunc(func(ctx context.Context, e Event) error {
		return next.HandleEvent(ctx, e)
	})
}

// asyncHandler simulates async work for throughput testing.
type asyncHandler struct {
	delay int // microseconds
}

func (h *asyncHandler) HandleEvent(ctx context.Context, e Event) error {
	if h.delay > 0 {
		// Simulate work without blocking the benchmark timer excessively
		var sum int
		for i := 0; i < h.delay*10; i++ {
			sum += i
		}
		_ = sum
	}
	return nil
}

// BenchmarkRouterWithWork measures realistic workload with handler execution time.
func BenchmarkRouterWithWork(b *testing.B) {
	router := NewRouter()
	ctx := context.Background()

	b.Run("NoWork", func(b *testing.B) {
		router.HandleFunc("work", noopHandler)
		event := &benchEvent{topic: "work"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})

	b.Run("LightWork", func(b *testing.B) {
		router.Handle("work", &asyncHandler{delay: 1}) // 1µs simulated work
		event := &benchEvent{topic: "work"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})

	b.Run("ModerateWork", func(b *testing.B) {
		router.Handle("work", &asyncHandler{delay: 10}) // 10µs simulated work
		event := &benchEvent{topic: "work"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Dispatch(ctx, event)
		}
	})
}

// BenchmarkRouterScaling measures how performance scales with route count.
func BenchmarkRouterScaling(b *testing.B) {
	routeCounts := []int{10, 50, 100, 500, 1000}

	for _, count := range routeCounts {
		b.Run(fmt.Sprintf("%dRoutes", count), func(b *testing.B) {
			router := NewRouter()
			ctx := context.Background()

			// Register routes
			for i := 0; i < count; i++ {
				pattern := fmt.Sprintf("route/%d/*", i)
				router.HandleFunc(pattern, noopHandler)
			}

			// Dispatch to middle route (worst case for linear search)
			event := &benchEvent{topic: fmt.Sprintf("route/%d/handler", count/2)}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				router.Dispatch(ctx, event)
			}
		})
	}
}
