// Package container defines a generic interface for managing containerized workloads.
//
// This package allows the lifecycle library to manage containers (Docker, Podman, etc.)
// without having a direct dependency on their respective SDKs. Consumers can implement
// the Container interface and pass it to a ContainerWorker for supervision.
//
// key features:
//   - **Container Interface**: Generic Start, Stop, Logs, and Status methods.
//   - **Mock Implementation**: A reference implementation for testing and local development.
package container
