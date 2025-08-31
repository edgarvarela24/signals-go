Of course. Since you've got a passing test and have finalized the API design, it's the perfect time to update the roadmap.

Here is the complete, revised `roadmap.md` file, updated to reflect the idiomatic, type-inferred `signals.New(scope, ...)` pattern for all reactive primitives. You can copy and paste this directly into your project.

-----

# Signals: Roadmap to v1.0 (Final)

> **Goal:** Build a production-grade, SolidJS-flavored reactive core for Go. The API is centered around an explicit `Engine` runtime and `Scope` objects, using idiomatic, package-level functions for creating primitives.
>
> ```go
> // Final API Goal
> eng := signals.Start()
> defer eng.Close()
> ```

> rootScope := eng.Scope()
> signal := signals.New(rootScope, 0) // Package-level function

> eng.Handler(func(s \*signals.Scope, w, r) {
> // Primitives are created by passing the request-specific scope
> reqSignal := signals.New(s, 1) // Dies with the request
> })
>
> ```
> 
> This plan is sliced into tiny, testable chunks, perfect for a TDD workflow and for learning Go incrementally. Each milestone ships something runnable and benchmarkable.
> ```

-----

## Table of Contents

1.  [Project Skeleton (Milestone 0)](https://www.google.com/search?q=%23project-skeleton-milestone-0)
2.  [Core: The `Engine` and Root `Scope` (M1)](https://www.google.com/search?q=%23core-the-engine-and-root-scope-m1)
3.  [Signals (M2)](https://www.google.com/search?q=%23signals-m2)
4.  [Effects (M3)](https://www.google.com/search?q=%23effects-m3)
5.  [Memos / Computed Values (M4)](https://www.google.com/search?q=%23memos--computed-values-m4)
6.  [Derived Scopes for Concurrency and Context (M5)](https://www.google.com/search?q=%23derived-scopes-for-concurrency-and-context-m5)
7.  [Resources (Async Fetching) (M6)](https://www.google.com/search?q=%23resources-async-fetching-m6)
8.  [Stores (Structured State) (M7)](https://www.google.com/search?q=%23stores-structured-state-m7)
9.  [Concurrency Options (M8)](https://www.google.com/search?q=%23concurrency-options-m8)
10. [Performance Pass + Benchmarks (M9)](https://www.google.com/search?q=%23performance-pass--benchmarks-m9)
11. [Reliability: Fuzzing, Race, Faults (M10)](https://www.google.com/search?q=%23reliability-fuzzing-race-faults-m10)
12. [Docs, Examples, Versioning (M11)](https://www.google.com/search?q=%23docs-examples-versioning-m11)
13. [Release Checklist (v1.0)](https://www.google.com/search?q=%23release-checklist-v10)
14. [Appendix: Final Public API Reference](https://www.google.com/search?q=%23appendix-final-public-api-reference)
15. [Appendix: Testing Tactics](https://www.google.com/search?q=%23appendix-testing-tactics)

-----

## Project Skeleton (Milestone 0)

**Outcome:** Clean repo with modules, layout, tooling, CI-ready.

```
signals/
  go.mod
  LICENSE
  README.md
  internal/
    sched/      # microtask queue & batching
    graph/      # cells, computations (internal core)
  pkg/
    signals/    # public API (this is the package users import)
  examples/
    counter/
    http-handler/
  bench/
    micro/
```

**Tasks**

  * `go mod init github.com/your-username/signals`
  * Set up `magefile` or `Makefile` with `test`, `bench`, `lint` targets.
  * Add a minimal "hello world" test that proves the test runner works.

**Definition of Done**

  * `go test ./...` passes.
  * CI config (GitHub Actions): run tests, race detector, `go vet`.

-----

## Core: The `Engine` and Root `Scope` (M1)

**Outcome:** An explicit reactive runtime (`Engine`) that can be started and stopped, and provides a main "root" `Scope`.

**Public API**

```go
package signals

func Start(opts ...Option) *Engine
func (e *Engine) Close() error
func (e *Engine) Scope() *Scope // Gets the main, long-lived scope

type Scope struct { /* unexported fields */ }
func (s *Scope) Batch(fn func()) // Batch is a core method of the scope's scheduler
```

**TDD Workflow**

1.  **Write `TestEngine_StartAndClose`**: Test that `Start()` returns a non-nil `*Engine` and `Close()` doesn't panic.
2.  **Write `TestEngine_Scope`**: Test that `eng.Scope()` returns a non-nil `*Scope`.
3.  **Refactor initial tests**: Adapt previous tests to use this new model. Create an engine, get its scope, then `Close()` the engine. Assert that a `Batch` call on the scope does nothing after the engine is closed.

**Definition of Done**

  * An `Engine` can be created and torn down.
  * It provides a primary `Scope` whose lifecycle is tied to the `Engine`.

-----

## Signals (M2)

**Outcome:** The core reactive primitive: a value cell that tracks subscriptions.

**Public API**

```go
// Package-level generic function
func New[T any](s *Scope, initial T) Signal[T]

// Interfaces
type Signal[T any] interface {
    Readonly[T] // Embeds Get()
    Set(T)
    Update(func(*T))
}

type Readonly[T any] interface {
    Get() T
}
```

**TDD Workflow**

1.  **Write `TestSignal_GetReturnsInitialValue`**: Call `New(scope, 10)` and assert that the returned signal's `Get()` method returns `10`.
2.  **Write `TestSignal_SetUpdatesValue`**: Call `Set(20)` and then assert `Get()` returns `20`.
3.  **Write `TestSignal_UpdateMutatesValue`**: Call `Update(func(v *int){ *v = 30 })` and assert `Get()` returns `30`.

**Definition of Done**

  * `New()` creates a stateful cell that can be read from and written to. The public API uses interfaces for good encapsulation.

-----

## Effects (M3)

**Outcome:** Side-effect computations that automatically react to signal changes. The full reactive loop is now complete.

**Public API**

```go
// Package-level generic functions
func Effect(s *Scope, fn func()) (cleanup func())
func Untrack(s *Scope, fn func())
func OnCleanup(s *Scope, fn func())
```

**TDD Workflow**

1.  **Write `TestEffect_RunsOnCreate`**: Create an effect and use a channel or a simple boolean to assert that its function runs once immediately.
2.  **Write `TestEffect_RerunsOnSignalChange`**:
      * Create a signal: `count := New(scope, 0)`.
      * Create a variable `runs := 0`.
      * Create an effect: `Effect(scope, func() { _ = count.Get(); runs++ })`.
      * Assert `runs` is `1`.
      * Call `count.Set(1)`.
      * Assert `runs` is `2`.
3.  **Write `TestBatching`**: Set a signal multiple times inside a `scope.Batch(...)` block and assert the effect only runs once after the block completes.
4.  **Write `TestUntrack`**: Call `Get()` on a signal inside an `Untrack(scope, ...)` block within an effect, and assert that setting that signal later does *not* cause the effect to re-run.
5.  **Write `TestOnCleanup`**: Use `OnCleanup` inside an effect. Trigger the effect to re-run and assert the cleanup function was called. Also test that `engine.Close()` triggers the final cleanup.

**Definition of Done**

  * Setting a signal now triggers dependent effects to run, respecting batching and untracking.
  * The reactive scheduler correctly orders updates.

-----

## Memos / Computed Values (M4)

**Outcome:** Derived, cached values that only re-run when dependencies *actually* change.

**Public API**

```go
// Package-level generic function
func Compute[T any](s *Scope, fn func() T, opts ...Option) Readonly[T]
```

**TDD Workflow**

1.  **Write `TestCompute_PropagatesChanges`**: Create a signal `a`, a computed `b` that returns `a.Get() * 2`, and an effect that reads `b`. Verify that changing `a` updates the effect.
2.  **Write `TestCompute_ShortCircuits`**:
      * Create a signal `num := New(scope, 0)`.
      * Create a computed `isEven := Compute(scope, func() bool { return num.Get() % 2 == 0 })`.
      * Create an effect that tracks how many times it runs based on `isEven`.
      * Call `num.Set(2)`. The effect should **not** re-run because the result of `isEven.Get()` (`true`) did not change.
      * Call `num.Set(3)`. The effect **should** re-run because the result is now `false`.

**Definition of Done**

  * `Compute()` creates a derived value that correctly caches its result and prevents unnecessary downstream updates.

-----

## Derived Scopes for Concurrency and Context (M5)

**Outcome:** A mechanism for creating temporary, cancellable scopes, perfect for HTTP requests or background jobs. Also includes a scope-bound DI mechanism.

**Public API**

```go
// Method on *Engine
func (e *Engine) NewScope(ctx context.Context) *Scope

// Package-level functions for DI
func Provide[T any](s *Scope, key any, value T)
func Use[T any](s *Scope, key any) (T, bool)
```

**TDD Workflow**

1.  **Write `TestDerivedScope_Cancellation`**:
      * Create an `Engine`.
      * Create a `context.WithCancel`.
      * Create a derived scope: `s := eng.NewScope(ctx)`.
      * Create an effect in the derived scope `s` that signals on a channel when it runs.
      * `cancel()` the context.
      * Verify that all work within the scope stops and it becomes disposed.
2.  **Write `TestProvideUse`**: Test the DI mechanism within a single scope and then with a derived scope to ensure values can be read from parent scopes.

**Definition of Done**

  * The `Engine` can create temporary scopes whose lifecycle is tied to a `context.Context`, ensuring no goroutine leaks.
  * The DI system provides a safe way to pass values down the scope tree.

-----

## Resources (Async Fetching) (M6)

**Outcome:** A declarative primitive for managing asynchronous data fetching, inspired by SolidJS `createResource`.

**Public API**

```go
// Package-level generic function
func Resource[K comparable, T any](
  s *Scope,
  key func() K,
  fetcher func(ctx context.Context, key K) (T, error),
  opts ...Option,
) Resource[T]

// Interface returned by Resource
type Resource[T any] interface {
    Readonly[T] // Embeds Get() which returns the latest successful value
    Loading() bool
    Error() error
}
```

**TDD Workflow**

1.  **Write `TestResource_FetchesOnCreate`**: Create a resource and verify its `Loading()` state is initially true, then becomes false, and `Get()` returns the fetched value.
2.  **Write `TestResource_RefetchesOnKeyChange`**: Create a signal for the resource key. Change the signal's value and assert that the fetcher function is called again.
3.  **Write `TestResource_CancelsOnKeyChange`**: In the fetcher, use a `time.Sleep` and check `ctx.Done()`. Change the key mid-fetch and assert that the original fetch's context is canceled.

**Definition of Done**

  * `Resource()` provides a declarative, cancellation-safe way to handle async operations.

-----

## Stores (Structured State) (M7)

**Outcome:** A convenient wrapper for managing nested or complex state objects reactively.

**Public API**

```go
// Package-level generic function
func Store[T any](s *Scope, initial T) Store[T]

// Interface for Store
type Store[T any] interface {
    Readonly[T] // Embeds Get()
    Set(T)
    Update(func(v *T))
}
```

**TDD Workflow**

1.  **Write `TestStore_BehavesLikeSignal`**: Create a store with a struct and verify that `Update` mutations trigger effects correctly.
2.  **Benchmark `Store_Update` vs `Signal_Set`**: For a large struct, verify that `Store.Update` results in fewer allocations than reading, modifying, and calling `Set` on a regular `Signal`.

**Definition of Done**

  * `Store()` provides an intuitive and efficient way to manage reactive structs.

-----

## Concurrency Options (M8)

**Outcome:** *Opt-in* cross-goroutine safety, allowing writes from any goroutine to be safely applied to the owning scope.

**Public API**

```go
// Functional option for Start
func WithConcurrent() Option

// Method on *Scope (only available on concurrent scopes)
func (s *Scope) Enqueue(fn func())
```

**TDD Workflow**

1.  **Write `TestConcurrent_SetFromGoroutine`**:
      * Start an engine with `signals.Start(signals.WithConcurrent())`.
      * Create a signal and an effect in the main goroutine.
      * In a new goroutine, use `scope.Enqueue(func() { signal.Set(...) })`.
      * Assert that the effect is updated correctly in the main goroutine without a data race.
2.  **Run all existing tests with `-race`** on a concurrent scope to ensure no race conditions were introduced.

**Definition of Done**

  * The engine can be configured to run in a concurrent mode.
  * `go test ./... -race` is consistently clean across all stress tests.

-----

## Performance Pass + Benchmarks (M9)

**Outcome:** A quantified understanding of the library's performance characteristics.

**Micro-benchmarks**

  * `SignalGet`: Aim for sub-10ns (near atomic load).
  * `SignalSet (no subscribers)`: Aim for sub-25ns.
  * `SignalSet (10 subscribers)`: Aim for sub-300ns.
  * `Compute (stable)`: The overhead for an unchanged computed value should be minimal (a version check).
  * `Resource (key flip)`: Time to cancel the old fetch and start the new one should be sub-20µs (excluding the fetch itself).

**Definition of Done**

  * Performance baselines are recorded and checked into the repository.
  * CI can optionally run benchmarks to detect major regressions.

-----

## Reliability: Fuzzing, Race, Faults (M10)

**Outcome:** Hardened library against edge cases, panics, and unexpected inputs.

**Tasks**

  * Implement runtime cycle detection (an effect that depends on itself, directly or indirectly) and panic with a helpful error.
  * Use Go's built-in fuzzing on the core primitives to discover unexpected interactions.
  * Write tests that inject panics into user-provided functions (in `Effect`, `Compute`) and assert that the reactive system remains stable and cleanup functions are still called.

**Definition of Done**

  * Fuzz tests are added to the CI pipeline.
  * The library is resilient to user code panics.
  * Cyclical dependencies are detected and reported clearly.

-----

## Docs, Examples, Versioning (M11)

**Outcome:** A new developer can understand the library's value and be productive within 5 minutes.

**Docs**

  * `README.md`: A comprehensive quickstart guide showing the `Engine` lifecycle and the `signals.New` pattern.
  * `docs/architecture.md`: A brief explanation of the `Engine`, `Scope`, and scheduler model.
  * `docs/concurrency.md`: A clear guide on when to use the default vs. concurrent engine.

**Examples**

  * `examples/counter`: A simple CLI app demonstrating `New`, `Compute`, `Effect`.
  * `examples/http-handler`: An example using `eng.Handler` and `signals.New(s, ...)` to manage request-specific state.
  * `examples/resource-search`: A web example using `Resource` with debouncing for a live search box.

**Definition of Done**

  * All public APIs are documented with GoDoc.
  * The README is clear, and all examples compile and run from a fresh clone.

-----

## Release Checklist (v1.0)

  * [ ] Public API is frozen and reflects the final design.
  * [ ] All API methods have comprehensive GoDoc documentation.
  * [ ] `go test ./... -race` is consistently clean.
  * [ ] Benchmarks are up-to-date with current performance numbers.
  * [ ] All examples in the `examples/` directory are working and well-commented.
  * [ ] `README.md` is complete and provides a compelling introduction.
  * [ ] `CHANGELOG.md` is created.
  * [ ] A `v1.0.0` Git tag is created and pushed.

-----

## Appendix: Final Public API Reference

```go
package signals

import "context"
import "net/http"

// --- Engine & Lifecycle ---

type Engine struct { /* unexported fields */ }
func Start(opts ...Option) *Engine
func (e *Engine) Close() error
func (e *Gngine) Scope() *Scope
func (e *Engine) NewScope(ctx context.Context) *Scope
func (e *Engine) Handler(h func(s *Scope, w http.ResponseWriter, r *http.Request)) http.HandlerFunc

// --- Scope ---

type Scope struct { /* unexported fields */ }
func (s *Scope) Batch(fn func())
// Concurrent scopes only
func (s *Scope) Enqueue(fn func())

// --- Primitives (Package-level functions) ---

func New[T any](s *Scope, initial T) Signal[T]
func Compute[T any](s *Scope, fn func() T, opts ...Option) Readonly[T]
func Effect(s *Scope, fn func()) (cleanup func())
func Store[T any](s *Scope, initial T) Store[T]
func Resource[K comparable, T any](s *Scope, key func() K, fetcher func(ctx context.Context, key K) (T, error), opts ...Option) Resource[T]

// --- Utilities (Package-level functions) ---

func Untrack(s *Scope, fn func())
func OnCleanup(s *Scope, fn func())

// --- DI (Package-level functions) ---

func Provide[T any](s *Scope, key any, value T)
func Use[T any](s *Scope, key any) (T, bool)

// --- Interfaces ---

type Signal[T any] interface {
    Readonly[T]
    Set(T)
    Update(func(*T))
}
type Readonly[T any] interface {
    Get() T
}
type Store[T any] interface {
    Signal[T]
}
type Resource[T any] interface {
    Readonly[T]
    Loading() bool
    Error() error
}
```

-----

## Appendix: Testing Tactics

  * **Unit Tests**: Use table-driven tests for different graph shapes (lines, fans, diamonds). Assert call order and counts by appending to a shared slice within a test's scope.
  * **Lifecycle Tests**: For every feature, add a test case that ensures it behaves correctly when the `Engine` is closed or a derived `Scope`'s context is canceled.
  * **Race Detector**: The `-race` flag is non-negotiable. Add a dedicated CI step that runs all tests under the race detector.
  * **Benchmarks**: Benchmark each primitive in isolation first, then in combination to understand the overhead of the scheduler and graph updates.
  * **Failure Injection**: Write tests that wrap user-provided functions (like in `Effect` or `Compute`) to deliberately panic. Assert that the system remains stable and doesn't crash.