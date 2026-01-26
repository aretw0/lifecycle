# lifecycle

[![Go Report Card](https://goreportcard.com/badge/github.com/aretw0/lifecycle)](https://goreportcard.com/report/github.com/aretw0/lifecycle)
[![Go Doc](https://godoc.org/github.com/aretw0/lifecycle?status.svg)](https://godoc.org/github.com/aretw0/lifecycle)
[![License](https://img.shields.io/github/license/aretw0/lifecycle.svg?color=red)](LICENSE.txt)
[![Release](https://img.shields.io/github/release/aretw0/lifecycle.svg?branch=main)](https://github.com/aretw0/lifecycle/releases)

**lifecycle** is a Go library for managing application shutdown signals and interactive terminal I/O robustly. It centralizes the "Dual Signal" logic and "Interruptible I/O" patterns originally extracted from [Trellis](https://github.com/aretw0/trellis) and designed for any tool needing robust signal handling.

## Vision

To provide a standard, leak-free way to handle CLI interruptions (Ctrl+C) and graceful shutdowns across Go CLI applications, handling OS idiosyncrasies (especially Windows `CONIN$`) transparently.

## Installation

```bash
go get github.com/aretw0/lifecycle
```

## Features

* **SignalContext**: Differentiates between `SIGINT` (User Interrupt) and `SIGTERM` (System Shutdown).
  * **SIGINT**: Captured but doesn't cancel context immediately (allows "Wait, are you sure?" logic).
  * **SIGTERM**: Cancels context immediately (standard graceful shutdown).
* **TermIO**:
  * **InterruptibleReader**: Wraps `io.Reader` to allow `Read()` calls to be abandoned when a context is cancelled (avoids goroutine leaks).
  * **Platform Aware**: Automatically uses `CONIN$` on Windows.
    * **Why?** On Windows, standard `os.Stdin` closes immediately upon receiving a signal (like Ctrl+C), causing a fatal `EOF` before the application can gracefully handle the signal. `lifecycle` switches to `CONIN$`, which keeps the handle open, allowing the `SignalContext` to process the event.
  * **UpgradeTerminal**: Helper to upgrade an arbitrary existing `io.Reader` (if it identifies as a terminal) to the safe platform-specific reader.

## Usage

### Signal Context

```go
package main

import (
    "context"
    "fmt"
    "github.com/aretw0/lifecycle"
)

func main() {
    // captures SIGINT/SIGTERM
    ctx := lifecycle.NewSignalContext(context.Background())
    defer ctx.Cancel() 

    <-ctx.Done()
    
    // Check why we stopped
    if sig := ctx.Signal(); sig != nil {
        fmt.Printf("Stopped by signal: %v\n", sig)
    }
}
```

### Interruptible I/O

```go
package main

import (
    "context"
    "fmt"
    "github.com/aretw0/lifecycle"
)

func main() {
    ctx := context.Background() // or SignalContext
    
    // Smart Open (handles Windows CONIN$)
    reader, _ := lifecycle.OpenTerminal()
    
    // Wrap to respect context cancellation
    r := lifecycle.NewInterruptibleReader(reader, ctx.Done())

    buf := make([]byte, 1024)
    n, err := r.Read(buf)
    if lifecycle.IsInterrupted(err) {
        fmt.Println("Read cancelled!")
        return
    }
    fmt.Printf("Read: %s\n", buf[:n])
}
```

## Documentation

* [**PRODUCT**](docs/PRODUCT.md): Vision & Mission.
* [**TECHNICAL**](docs/TECHNICAL.md): Architecture & Design (with Diagrams).
* [**PLANNING**](docs/PLANNING.md): Roadmap & Backlog.
