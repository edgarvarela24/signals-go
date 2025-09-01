# Signals & RDS: Roadmap to v1.0 (Final)

 **Goal:** Build a production-grade, SolidJS-flavored reactive core for Go (`signals`), and a companion library of high-performance reactive data structures (`rds`). The API is centered around an explicit `Engine` runtime and `Scope` objects, using idiomatic, package-level functions.

 ```go
 // Final API Goal
 eng := signals.Start()
 defer eng.Close()
 rootScope := eng.Scope()
 ```

 // Core primitive
 counter := signals.New(rootScope, 0)

 // Reactive Data Structure
 users := rds.NewMap[suspicious link removed]
 users.Set(1, User{Name: "Alice"})

 // Derived view that stays in sync automatically
 alice := users.Get(1) // signals.Readonly[User]

 ```
 
 This plan is sliced into tiny, testable chunks, perfect for a TDD workflow. Each milestone ships something runnable and benchmarkable.
 ```

-----

## Table of Contents

1.  [Project Skeleton (Milestone 0)](https://www.google.com/search?q=%23project-skeleton-milestone-0)
2.  [Core: The `Engine` and Root `Scope` (M1)](https://www.google.com/search?q=%5Bhttps://www.google.com/search%3Fq%3D%2523core-the-engine-and-root-scope-m1%5D\(https://www.google.com/search%3Fq%3D%2523core-the-engine-and-root-scope-m1\))
3.  [Signals (M2)](https://www.google.com/search?q=%23signals-m2)
4.  [Effects (M3)](https://www.google.com/search?q=%23effects-m3)
5.  [Memos / Computed Values (M4)](https://www.google.com/search?q=%23memos--computed-values-m4)
6.  [**`rds` v0.1: The `ReactiveMap` (M5)**](https://www.google.com/search?q=%5Bhttps://www.google.com/search%3Fq%3D%2523rds-v01-the-reactivemap-m5%5D\(https://www.google.com/search%3Fq%3D%2523rds-v01-the-reactivemap-m5\))
7.  [**`rds` v0.2: The `ReactiveList` (M6)**](https://www.google.com/search?q=%5Bhttps://www.google.com/search%3Fq%3D%2523rds-v02-the-reactivelist-m6%5D\(https://www.google.com/search%3Fq%3D%2523rds-v02-the-reactivelist-m6\))
8.  [Derived Scopes for Concurrency and Context (M7)](https://www.google.com/search?q=%23derived-scopes-for-concurrency-and-context-m7)
9.  [Resources (Async Fetching) (M8)](https://www.google.com/search?q=%23resources-async-fetching-m8)
10. [Stores (Structured State) (M9)](https://www.google.com/search?q=%23stores-structured-state-m9)
11. [Concurrency Options (M10)](https://www.google.com/search?q=%23concurrency-options-m10)
12. [Performance Pass + Benchmarks (M11)](https://www.google.com/search?q=%23performance-pass--benchmarks-m11)
13. [Reliability: Fuzzing, Race, Faults (M12)](https://www.google.com/search?q=%23reliability-fuzzing-race-faults-m12)
14. [Docs, Examples, Versioning (M13)](https://www.google.com/search?q=%23docs-examples-versioning-m13)
15. [Release Checklist (v1.0)](https://www.google.com/search?q=%23release-checklist-v10)
16. [Appendix: Final Public API Reference](https://www.google.com/search?q=%23appendix-final-public-api-reference)
17. [Appendix: Testing Tactics](https://www.google.com/search?q=%23appendix-testing-tactics)

-----

## Project Skeleton (Milestone 0)

**Outcome:** Clean repo with modules, layout, tooling, CI-ready.

```
signals/
  go.mod
  LICENSE
  README.md
  internal/
    ...
  pkg/
    signals/    # Public API for the core reactive primitives
    rds/        # Public API for reactive data structures
  examples/
    ...
  bench/
    ...
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
func (e *Engine) Scope() *Scope

type Scope struct { /* unexported fields */ }
func (s *Scope) Batch(fn func())
```

**Definition of Done**

  * An `Engine` can be created and torn down.
  * It provides a primary `Scope` whose lifecycle is tied to the `Engine`.

-----

## Signals (M2)

**Outcome:** The core reactive primitive: a value cell that tracks subscriptions.

**Public API**

```go
package signals

func New[T any](s *Scope, initial T) Signal[T]

type Signal[T any] interface {
    Readonly[T]
    Set(T)
    Update(func(*T))
}

type Readonly[T any] interface {
    Get() T
}
```

**Definition of Done**

  * `New()` creates a stateful cell that can be read from and written to.

-----

## Effects (M3)

**Outcome:** Side-effect computations that automatically react to signal changes. The full reactive loop is now complete.

**Public API**

```go
package signals

func Effect(s *Scope, fn func()) (cleanup func())
func Untrack(s *Scope, fn func())
func OnCleanup(s *Scope, fn func())
```

**Definition of Done**

  * Setting a signal now triggers dependent effects to run, respecting batching and untracking.

**TDD Workflow**

1.  **`TestEffect_RunsOnSignalChanges`**: Create a signal and an effect that reads it. Change the signal's value and assert the effect runs.
2.  **`TestEffect_OnlyRunsOnDependentSignalChanges`**: Create two signals (`a`, `b`) and an effect that only reads `a`. Change `b` and assert the effect does *not* run.
3.  **`TestEffect_BatchedChangesRunOnce`**: In a `scope.Batch()` call, set a signal multiple times. Assert the dependent effect runs only once.
4.  **`TestEffect_UntrackPreventsDependencies`**: Inside an effect, read one signal normally and another inside an `Untrack()` block. Assert that only changes to the first signal trigger the effect.
5.  **`TestEffect_OnCleanupIsCalled`**: Create an effect and register a cleanup function using `OnCleanup`. When the scope is disposed, assert the cleanup function was called.
6.  **`TestEffect_ReturnedCleanupStopsEffect`**: Call the `cleanup` function returned by `Effect`. Then, change a dependency and assert the effect does *not* run.

-----

## Memos / Computed Values (M4)

**Outcome:** Derived, cached values that only re-run when dependencies *actually* change value.

**Public API**

```go
package signals

func Compute[T any](s *Scope, fn func() T, opts ...Option) Readonly[T]
```

**TDD Workflow**

1.  **Write `TestCompute_PropagatesChanges`**: Create a signal `a`, a computed `b` that returns `a.Get() * 2`, and an effect that reads `b`. Verify that changing `a` updates the effect.
2.  **Write `TestCompute_ShortCircuits`**:
      * Create `num := New(scope, 0)`.
      * Create `isEven := Compute(scope, func() bool { return num.Get() % 2 == 0 })`.
      * Create an effect tracking `isEven`.
      * Call `num.Set(2)`. The effect should **not** re-run because the result of `isEven.Get()` (`true`) did not change.

**Definition of Done**

  * `Compute()` creates a derived value that correctly caches its result and prevents unnecessary downstream updates. **This is the final primitive needed to begin `rds`.**

-----

## `rds` v0.1: The `ReactiveMap` (M5)

**Outcome:** The first reactive data structure, providing fine-grained, key-based reactivity. This milestone serves as the primary dogfooding exercise for the core `signals` API.

**Public API**

```go
package rds

func NewMap[K comparable, V any](s *signals.Scope) ReactiveMap[K, V]

type ReactiveMap[K comparable, V any] interface {
    Size() signals.Readonly[int]
    Get(key K) signals.Readonly[V]
    Has(key K) signals.Readonly[bool]
    Set(key K, value V)
    Delete(key K)
}
```

**TDD Workflow**

1.  **Write `TestReactiveMap_GetAndSet`**: Create an effect that depends on `m.Get("foo")`. Assert it runs. Call `m.Set("foo", 123)` and assert the effect runs again.
2.  **Write `TestReactiveMap_StableReruns`**: The "wow" test. Create effects for keys "a" and "b". In a batch, update "a" and set "b" to its *existing* value. Assert only the effect for "a" re-ran. This proves per-key dependency tracking and value-based short-circuiting.
3.  **Write `TestReactiveMap_SizeAndDelete`**: Ensure that mutations correctly update the reactive `Size()` signal.

**Definition of Done**

  * A working `ReactiveMap` that correctly tracks dependencies on a per-key basis and serves as a strong validation of the M2-M4 primitives.

-----

## `rds` v0.2: The `ReactiveList` (M6)

**Outcome:** A list data structure optimized for reactive updates using a chunk-based tokenization strategy to ensure high performance with large collections.

**Public API**

```go
package rds

func NewList[T any](s *signals.Scope, initial ...T) ReactiveList[T]

type ReactiveList[T any] interface {
    Len() signals.Readonly[int]
    At(index int) signals.Readonly[T]
    Set(index int, v T)
    Push(v T)
    Delete(index int)
}
```

**TDD Workflow**

1.  **Write `TestReactiveList_AtAndSet`**: Create an effect that depends on `list.At(5)`. Call `list.Set(5, ...)` and assert the effect runs again.
2.  **Write `TestReactiveList_ChunkedInvalidation`**: Create effects for `list.At(5)` and `list.At(500)` (assuming a chunk size of \< 495). Call `list.Set(5, ...)`. Assert the first effect ran but the second **did not**. This proves the chunking optimization.
3.  **Write `TestReactiveList_PushAndLen`**: Test mutations that affect the list's length and verify the `Len()` signal updates correctly.

**Definition of Done**

  * A performant `ReactiveList` that avoids excessive updates for localized changes.

-----

## Derived Scopes for Concurrency and Context (M7)

**Outcome:** A mechanism for creating temporary, cancellable scopes. Crucial for managing state in scenarios like HTTP handlers, where you might create a request-scoped `ReactiveMap`.

**Public API**

```go
package signals

func (e *Engine) NewScope(ctx context.Context) *Scope
```

**Definition of Done**

  * The `Engine` can create temporary scopes whose lifecycle is tied to a `context.Context`, ensuring no goroutine leaks.

-----

## Resources (Async Fetching) (M8)

**Outcome:** A declarative primitive for managing asynchronous data fetching.

**Public API**

```go
package signals

func Resource[K comparable, T any](
  s *Scope,
  key func() K,
  fetcher func(ctx context.Context, key K) (T, error),
) Resource[T]

type Resource[T any] interface {
    Readonly[T]
    Loading() bool
    Error() error
}
```

**Definition of Done**

  * `Resource()` provides a declarative, cancellation-safe way to handle async operations.

-----

## Stores (Structured State) (M9)

**Outcome:** A convenient wrapper for managing nested or complex state objects reactively. This serves as an alternative to `rds` for simpler, monolithic state blocks.

**Public API**

```go
package signals

func Store[T any](s *Scope, initial T) Store[T]

type Store[T any] interface {
    Signal[T]
}
```

**Definition of Done**

  * `Store()` provides an intuitive way to manage reactive structs, offering a coarser-grained reactivity model compared to `rds`.

-----

## Concurrency Options (M10)

**Outcome:** *Opt-in* cross-goroutine safety, allowing writes to the `Engine`'s main goroutine from anywhere.

**Public API**

```go
package signals

func WithConcurrent() Option
func (s *Scope) Enqueue(fn func())
```

**Definition of Done**

  * The engine can be configured to run in a concurrent mode where `go test ./... -race` is consistently clean.

-----

## Performance Pass + Benchmarks (M11)

**Outcome:** A quantified understanding of the performance characteristics for both `signals` and `rds`.

**Benchmarks**

  * **`signals`**: `SignalGet`, `SignalSet`, `Compute (stable)`.
  * **`rds`**: `ReactiveMap.Get (stable)`, `ReactiveMap.Set`, `ReactiveList.At (stable)`, `ReactiveList.Set (in-chunk)`.

**Definition of Done**

  * Performance baselines are recorded and checked into the repository.

-----

## Reliability: Fuzzing, Race, Faults (M12)

**Outcome:** Hardened library against edge cases, panics, and unexpected inputs.

**Tasks**

  * Implement runtime cycle detection.
  * Use Go's built-in fuzzing on core primitives and `rds` methods.
  * Inject panics into user-provided functions to assert system stability.

**Definition of Done**

  * Fuzz tests are added to the CI pipeline. Cyclical dependencies are detected.

-----

## Docs, Examples, Versioning (M13)

**Outcome:** A new developer can be productive within 5 minutes.

**Docs & Examples**

  * `README.md`: Quickstart showing `signals.New` and `rds.NewMap`.
  * `examples/http-leaderboard`: An example using a request-scoped `ReactiveList` to show a live, sorted view of data.

**Definition of Done**

  * All public APIs are documented. Examples compile and run.

-----

## Release Checklist (v1.0)

  * [ ] Public API is frozen.
  * [ ] All API methods have comprehensive GoDoc.
  * [ ] `go test ./... -race` is clean.
  * [ ] Benchmarks are up-to-date.
  * [ ] All examples are working and well-commented.
  * [ ] `README.md` is complete.
  * [ ] `CHANGELOG.md` is created.
  * [ ] A `v1.0.0` Git tag is created and pushed.

-----

## Appendix: Final Public API Reference

```go
// --- Package: signals ---

// Engine & Lifecycle
func Start(opts ...Option) *Engine
func (e *Engine) Close() error
func (e *Engine) Scope() *Scope
func (e *Engine) NewScope(ctx context.Context) *Scope

// Scope
type Scope struct { /* ... */ }
func (s *Scope) Batch(fn func())
func (s *Scope) Enqueue(fn func()) // Concurrent scopes only

// Primitives
func New[T any](s *Scope, initial T) Signal[T]
func Compute[T any](s *Scope, fn func() T) Readonly[T]
func Effect(s *Scope, fn func()) (cleanup func())
func Store[T any](s *Scope, initial T) Store[T]
func Resource[K comparable, T any](s *Scope, ...) Resource[T]

// Utilities
func Untrack(s *Scope, fn func())
func OnCleanup(s *Scope, fn func())

// --- Package: rds ---

// Reactive Data Structures
func NewMap[K comparable, V any](s *signals.Scope) ReactiveMap[K, V]
func NewList[T any](s *signals.Scope, initial ...T) ReactiveList[T]

// Interfaces
type ReactiveMap[K comparable, V any] interface { /* Get, Set, etc. */ }
type ReactiveList[T any] interface { /* At, Set, etc. */ }
```

-----

## Appendix: Testing Tactics

  * **Unit Tests**: Use table-driven tests. Assert call counts and order.
  * **Lifecycle Tests**: Ensure all primitives and `rds` structures clean up correctly when a `Scope` is cancelled.
  * **Invalidation Tests**: For `rds`, write specific tests to confirm the invalidation boundaries (per-key vs. per-chunk) are respected.
  * **Race Detector**: The `-race` flag is non-negotiable.
  * **Failure Injection**: Wrap user functions to deliberately panic and assert system stability.