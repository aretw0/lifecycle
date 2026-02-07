# BaseWorker Example

This example demonstrates the **BaseWorker embedding pattern** to eliminate repetitive boilerplate code.

## The Problem

Every custom worker needs to implement 5+ methods, most of which are identical:

```go
type MyWorker struct {
    done chan error  // Manual management
}

func (w *MyWorker) Start(ctx context.Context) error {
    w.done = make(chan error, 1)
    go func() {
        w.done <- w.Run(ctx)
        close(w.done)
    }()
    return nil
}

func (w *MyWorker) Stop(ctx context.Context) error { return nil }
func (w *MyWorker) Wait() <-chan error             { return w.done }
func (w *MyWorker) String() string                 { return "MyWorker" }
func (w *MyWorker) State() lifecycle.WorkerState {
    return lifecycle.WorkerState{Name: "MyWorker"}
}
```

**Total**: ~15 lines of repetitive boilerplate!

## The Solution: BaseWorker

Embed `lifecycle.BaseWorker` to get default implementations:

```go
type MyWorker struct {
    lifecycle.BaseWorker  // ← All boilerplate methods provided!
    // Your custom fields here
}

func NewMyWorker() *MyWorker {
    return &MyWorker{
        BaseWorker: *lifecycle.NewBaseWorker("MyWorker"),
    }
}

func (w *MyWorker) Start(ctx context.Context) error {
    return w.StartFunc(ctx, w.Run)  // ← Helper manages goroutine + done channel
}

func (w *MyWorker) Run(ctx context.Context) error {
    // Your business logic here
    return nil
}
```

**Total**: ~2 lines of setup code!

## What You Get

BaseWorker provides:

- ✅ `Stop(ctx)` — no-op (context cancellation handles cleanup)
- ✅ `Wait()` — returns done channel
- ✅ `String()` — returns worker name
- ✅ `State()` — returns minimal state with name
- ✅ `StartFunc(ctx, fn)` — helper for common goroutine pattern

## Running the Example

```bash
go run examples/baseworker/main.go
```

You'll see:

```log
=== BaseWorker Example ===
Demonstrating reduced boilerplate with embedding pattern

Worker name: SimpleWorker
Worker state: {Name:SimpleWorker}

INFO SimpleWorker tick message="Supervised worker!"
```

Press Ctrl+C to exit gracefully.

## Comparison

| Aspect | Old Approach | BaseWorker |
|:---|:---|:---|
| **Lines of boilerplate** | ~15 lines | ~2 lines |
| **Manual channel mgmt** | Yes | No (StartFunc handles it) |
| **Override flexibility** | Full | Full (can override any method) |
| **Learning curve** | Low | Low (standard Go embedding) |

## When to Use

✅ **Use BaseWorker** when building simple workers  
✅ **Use BaseWorker** to reduce boilerplate  
⚠️ **Override methods** if you need custom behavior (e.g., rich State)  
⚠️ **Don't use** if worker has complex lifecycle (but rare)

## Related Examples

- **[suspend](../suspend/)** - Could benefit from BaseWorker (Generator, Worker, Blocker)
- **[reliability](../reliability/)** - Shows manual approach for comparison

---

**Pro Tip**: All existing workers keep working! BaseWorker is opt-in.
