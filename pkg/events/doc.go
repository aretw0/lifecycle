// Package events implements the Lifecycle Control Plane.
//
// It provides a reactive, event-driven orchestration layer that decouples
// event production (Sources) from business logic reactions (Handlers)
// using a centralized dispatcher (Router).
//
// # Core Concepts
//
//   - Router: The central hub that collects events from multiple sources and
//     dispatches them to matching handlers based on pattern matching (glob).
//   - Source: Event producers that bridge external stimuli (OS Signals,
//     Webhooks, File Changes, Tickers) into the lifecycle ecosystem.
//   - Handler: Components that react to events to perform actions like
//     reloading configuration, suspending workers, or initiating shutdown.
//   - Event: A simple string-based stimulus that triggers reactions.
//
// # Usage Pattern
//
//	router := events.NewRouter()
//
//	// 1. Add Sources
//	router.AddSource(events.NewWebhookSource(":8080"))
//	router.AddSource(events.NewFileWatchSource("config.yaml"))
//
//	// 2. Register Handlers
//	router.Handle("webhook/reload", events.NewReloadHandler(myReloadFunc))
//	router.HandleFunc("file/modified", func(ctx context.Context, e events.Event) error {
//	    return myCustomAction()
//	})
//
//	// 3. Run with Lifecycle
//	lifecycle.Run(router)
//
// # Design Philosophy
//
// The Control Plane is designed to be "plug-and-play". By consolidating all
// event logic here, applications can easily swap a "Signal-based reload"
// for a "Webhook-based reload" without changing the core business logic.
//
// # Context-Aware Handlers
//
// Handlers like NewShutdown and NewShutdownFunc are context-aware; they
// automatically discover the active signal.Context using signal.FromContext(ctx)
// and trigger its Shutdown() method. This ensures that even in complex
// interactive modes, the "Quit" command correctly exits the main Run loop.
package events
