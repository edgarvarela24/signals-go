# Signals: Roadmap to v1.0 (Revised)

 **Goal:** Build a production-grade, SolidJS-flavored reactive core for Go. The API is centered around an explicit `Engine` runtime and `Scope` objects that own all reactive primitives.

 ```go
 // Final API Goal
 eng := signals.Start()
 defer eng.Close()

 rootScope := eng.Scope()
 signal := rootScope.New(0)

 eng.Handler(func(s \*signals.Scope, w, r) {
 reqSignal := s.New(1) // Dies with the request
 })

```

This plan is sliced into tiny, testable chunks, perfect for a TDD workflow and for learning Go incrementally. Each milestone ships something runnable and benchmarkable.


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
func (s *Scope) Batch(fn func()) // For now, just handles disposed state
```

**TDD Workflow**

1.  **Write `TestEngine_StartAndClose`**: Test that `Start()` returns a non-nil `*Engine` and `Close()` doesn't panic.
2.  **Write `TestEngine_Scope`**: Test that `eng.Scope()` returns a non-nil `*Scope`.
3.  **Refactor your existing tests**: Adapt `TestRoot_DisposePreventsFutureWork` to use this new model. Create an engine, get its scope, then `Close()` the engine. Assert that a `Batch` call on the scope does nothing.

**Definition of Done**

  * An `Engine` can be created and torn down.
  * It provides a primary `Scope` whose lifecycle is tied to the `Engine`.

-----

## Signals (M2)

**Outcome:** The core reactive primitive: a value cell that tracks subscriptions.

**Public API**

```go
// Method on *Scope
func (s *Scope) New[T any](initial T) Signal[T]

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

1.  **Write `TestSignal_GetReturnsInitialValue`**: Call `scope.New(10)` and assert that the returned signal's `Get()` method returns `10`.
2.  **Write `TestSignal_SetUpdatesValue`**: Call `Set(20)` and then assert `Get()` returns `20`.
3.  **Write `TestSignal_UpdateMutatesValue`**: Call `Update(func(v *int){ *v = 30 })` and assert `Get()` returns `30`.

**Definition of Done**

  * `scope.New()` creates a stateful cell that can be read from and written to. The public API uses interfaces for good encapsulation.

-----

## Effects (M3)

**Outcome:** Side-effect computations that automatically react to signal changes. The full reactive loop is now complete.

**Public API**

```go
// Method on *Scope
func (s *Scope) Effect(fn func()) (cleanup func())
func (s *Scope) Untrack(fn func())
func (s *Scope) OnCleanup(fn func())
```

**TDD Workflow**

1.  **Write `TestEffect_RunsOnCreate`**: Create an effect and use a channel or a simple boolean to assert that its function runs once immediately.
2.  **Write `TestEffect_RerunsOnSignalChange`**:
      * Create a signal: `count := scope.New(0)`.
      * Create a variable `runs := 0`.
      * Create an effect: `scope.Effect(func() { _ = count.Get(); runs++ })`.
      * Assert `runs` is `1`.
      * Call `count.Set(1)`.
      * Assert `runs` is `2`.
3.  **Write `TestBatching`**: Set a signal multiple times inside a `scope.Batch(...)` block and assert the effect only runs once after the block completes.
4.  **Write `TestUntrack`**: Call `Get()` on a signal inside an `scope.Untrack(...)` block within an effect, and assert that setting that signal later does *not* cause the effect to re-run.
5.  **Write `TestOnCleanup`**: Use `OnCleanup` inside an effect. Trigger the effect to re-run and assert the cleanup function was called. Also test that `engine.Close()` triggers the final cleanup.

**Definition of Done**

  * Setting a signal now triggers dependent effects to run, respecting batching and untracking.
  * The reactive scheduler correctly orders updates.

-----

## Memos / Computed Values (M4)

**Outcome:** Derived, cached values that only re-run when dependencies *actually* change.

**Public API**

```go
// Method on *Scope
func (s *Scope) Compute[T any](fn func() T, opts ...Option) Readonly[T]
```

**TDD Workflow**

1.  **Write `TestCompute_PropagatesChanges`**: Create a signal `a`, a computed `b` that returns `a.Get() * 2`, and an effect that reads `b`. Verify that changing `a` updates the effect.
2.  **Write `TestCompute_ShortCircuits`**:
      * Create a signal `num := scope.New(0)`.
      * Create a computed `isEven := scope.Compute(func() bool { return num.Get() % 2 == 0 })`.
      * Create an effect that tracks how many times it runs based on `isEven`.
      * Call `num.Set(2)`. The effect should **not** re-run because the result of `isEven.Get()` (`true`) did not change.
      * Call `num.Set(3)`. The effect **should** re-run because the result is now `false`.

**Definition of Done**

  * `scope.Compute` creates a derived value that correctly caches its result and prevents unnecessary downstream updates.

-----

## Derived Scopes for Concurrency and Context (M5)

**Outcome:** A mechanism for creating temporary, cancellable scopes, perfect for HTTP requests or background jobs. Also includes a scope-bound DI mechanism.

**Public API**

```go
// Method on *Engine
func (e *Engine) NewScope(ctx context.Context) *Scope

// Methods on *Scope for DI
func (s *Scope) Provide[T any](key any, value T)
func (s *Scope) Use[T any](key any) (T, bool)
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
// Method on *Scope
func (s *Scope) Resource[K comparable, T any](
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

  * `scope.Resource` provides a declarative, cancellation-safe way to handle async operations.

-----

## Stores (Structured State) (M7)

**Outcome:** A convenient wrapper for managing nested or complex state objects reactively.

**Public API**

```go
// Method on *Scope
func (s *Scope) Store[T any](initial T) Store[T]

// Interface for Store
type Store[T any] interface {
    Readonly[T] // Embeds Get()
    Set(T)
    Update(func(v *T))
}
```

**Behavior**

  * This is primarily an ergonomic wrapper around `scope.New`. It provides the same `Signal` interface but might be optimized for pointer-based updates on complex structs.

**TDD Workflow**

1.  **Write `TestStore_BehavesLikeSignal`**: Create a store with a struct and verify that `Update` mutations trigger effects correctly.
2.  **Benchmark `Store_Update` vs `Signal_Set`**: For a large struct, verify that `Store.Update` results in fewer allocations than reading, modifying, and calling `Signal.Set`.

**Definition of Done**

  * `scope.Store` provides an intuitive and efficient way to manage reactive structs.

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

**TDD Workflow**

1.  Create `*_test.go` files inside a `bench/` directory.
2.  Use `testing.B` and `b.ReportAllocs()` to write benchmarks for each primitive.
3.  Target "steady-state zero allocs" for all `Get()` methods.

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

  * `README.md`: A comprehensive quickstart guide showing the `Engine` lifecycle and a simple reactive example.
  * `docs/architecture.md`: A brief explanation of the `Engine`, `Scope`, and scheduler model.
  * `docs/concurrency.md`: A clear guide on when to use the default vs. concurrent engine.

**Examples**

  * `examples/counter`: A simple CLI app demonstrating `New`, `Compute`, `Effect`.
  * `examples/http-handler`: An example using `eng.Handler` and `NewScope` to manage request-specific state.
  * `examples/resource-search`: A web example using `scope.Resource` with debouncing for a live search box.

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

// --- Engine & Lifecycle ---

type Engine struct { /* unexported fields */ }
func Start(opts ...Option) *Engine
func (e *Engine) Close() error
func (e *Engine) Scope() *Scope
func (e *Engine) NewScope(ctx context.Context) *Scope
func (e *Engine) Handler(h func(s *Scope, w http.ResponseWriter, r *http.Request)) http.HandlerFunc

// --- Scope ---

type Scope struct { /* unexported fields */ }

// Primitives are created from a Scope
func (s *Scope) New[T any](initial T) Signal[T]
func (s *Scope) Compute[T any](fn func() T, opts ...Option) Readonly[T]
func (s *Scope) Effect(fn func()) (cleanup func())
func (s *Scope) Store[T any](initial T) Store[T]
func (s *Scope) Resource[K comparable, T any](key func() K, fetcher func(ctx context.Context, key K) (T, error), opts ...Option) Resource[T]

// Utilities on Scope
func (s *Scope) Batch(fn func())
func (s *Scope) Untrack(fn func())
func (s *Scope) OnCleanup(fn func())

// DI on Scope
func (s *Scope) Provide[T any](key any, value T)
func (s *Scope) Use[T any](key any) (T, bool)

// Concurrency on Scope (only on concurrent scopes)
func (s *Scope) Enqueue(fn func())

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