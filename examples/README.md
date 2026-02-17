# lifecycle Examples

This folder contains runnable examples demonstrating various features of the `lifecycle` library.

## Basic Patterns (CLI & Automation)

These examples demonstrate the core "Death Management" features (v1.0-v1.4). Ideal for CLIs, Scripts, and simple Tools.

* [**basic**](./basic/main.go): The "Hello World" of `lifecycle`. Shows `Run`, `Go`, and `Job`.
* [**context**](./context/main.go): Manual setup via `lifecycle.Context()` for gradual migration.
* [**hooks**](./hooks/main.go): How to register and execute synchronous/asynchronous cleanup hooks.
* [**interactive_dx**](./interactive_dx/main.go): Safe reading from Stdin (Windows `CONIN$` support) that respects context cancellation.

## Advanced Patterns (Control Plane v1.5+)

These examples demonstrate "Life Management" capabilities (v1.5+). Ideal for Services, Daemons, and Agents.

* [**suspend**](./suspend/main.go): The full Control Plane experience. Shows Supervisors, Suspend/Resume events, and Durable Execution.
* [**supervisor**](./supervisor/main.go): Managing a tree of child processes/workers with restart policies.
* [**repl**](./repl/main.go): Building an interactive REPL that handles signals and custom commands.

## Recipes

Specific solutions to common problems.

* [**zombie**](./zombie/README.md): Demonstrating Process Hygiene (Job Objects / PDeathSig) to ensure child processes die with the parent.
* [**reliability**](./reliability/main.go): Using `lifecycle.Do` for safe execution.
* [**observability**](./observability/main.go): Visualizing the system state with Mermaid.
