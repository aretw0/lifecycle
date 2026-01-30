// Package supervisor implements the Supervisor Pattern for managing process and worker lifecycles.
//
// A Supervisor is a special type of Worker that manages a collection of child Workers.
// It is responsible for starting (spawning), monitoring (watching), and restarting (healing)
// its children based on defined strategies.
//
// # Strategies
//
// The supervisor supports the following restart strategies:
//
//   - StrategyOneForOne: If a child process terminates, only that process is restarted.
//   - StrategyOneForAll: If a child process terminates, all other child processes are terminated,
//     and then all child processes are restarted.
//
// Usage
//
//	sup := supervisor.New("my-sup", supervisor.StrategyOneForOne,
//	    supervisor.Spec{
//	        Name: "worker-1",
//	        Factory: func() (worker.Worker, error) {
//	            return worker.NewProcess("worker-1", "sleep", "10"), nil
//	        },
//	    },
//	)
//
//	sup.Start(ctx)
package supervisor
