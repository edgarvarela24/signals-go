# Signals: Roadmap to v1.0

> **Goal:** Build a production-grade, SolidJS-flavored reactive core for Go with an explicit `Scope` and a clean public API:
>
> ```go
> get, set := CreateSignal($, initial)
> memo     := CreateMemo($, fn)
> CreateEffect($, fn)
> Batch($, fn)
> Untrack($, fn)
> ```
>
> Signals return **getter/setter functions** (e.g., `firstName()` / `setFirstName("…")`) rather than method handles.

You’re a solo dev, learning Go while you build this. This plan is sliced into **tiny, testable chunks**. Every milestone ships something runnable + benchmarkable, and each phase defines “Definition of Done”.

---

## Table of Contents

1. [Project Skeleton (Milestone 0)](#project-skeleton-milestone-0)
2. [Core: Scope + Scheduler (M1)](#core-scope--scheduler-m1)
3. [Signals (M2)](#signals-m2)
4. [Effects + Batching + Untrack (M3)](#effects--batching--untrack-m3)
5. [Memos (M4)](#memos-m4)
6. [Context: Provide/Use (M5)](#context-provideuse-m5)
7. [Resources (async keyed fetch) (M6)](#resources-async-keyed-fetch-m6)
8. [Stores (structured state) (M7)](#stores-structured-state-m7)
9. [Concurrency Options (M8)](#concurrency-options-m8)
10. [Performance Pass + Benchmarks (M9)](#performance-pass--benchmarks-m9)
11. [Reliability: Fuzzing, Race, Faults (M10)](#reliability-fuzzing-race-faults-m10)
12. [Docs, Examples, Versioning (M11)](#docs-examples-versioning-m11)
13. [Release Checklist (v1.0)](#release-checklist-v10)
14. [Appendix: Naming & Public API Reference](#appendix-naming--public-api-reference)
15. [Appendix: Testing Tactics](#appendix-testing-tactics)

---

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
    http-live-config/
  bench/
    micro/
```

**Tasks**

* `go mod init github.com/you/signals`
* Set up `make test`, `make bench`, `make lint`.
* Add a tiny “hello world” test exercising `Root` with a no-op.

**Definition of Done**

* `go test ./...` passes (even if mostly empty).
* CI config (GitHub Actions): run tests, race detector, `go vet`.

---

## Core: Scope + Scheduler (M1)

**Outcome:** Deterministic execution boundary + microtask queue.

**Implement**

* `func Root(fn func($ Scope)) (dispose func())`
* `type Scope interface { Batch(fn func()); Untrack[T any](fn func() T) T }`
* Internal microtask queue (`sched`): enqueue, drain, prevent re-entrancy.

**Notes**

* **Single goroutine assumption** for now: no locks on the hot path.
* `dispose()` tears down the scope and cancels pending tasks.

**Tests**

* Root runs, dispose stops future tasks.
* Batch executes the body and then flushes queued tasks once.

**Definition of Done**

* `Root/Batch/Untrack` exist with tests proving order guarantees.

---

## Signals (M2)

**Outcome:** First-class reactive value cell with getter/setter closures.

**Public API**

```go
func CreateSignal[T any]($ Scope, initial T) (
  get func() T,
  set func(T),
  update func(func(*T)), // pointer-mutator to avoid copies
)
```

**Internal**

* Cell holds `value`, `version uint64`, `subs` (subscriber IDs).
* `get()` registers dependency when a computation is active.
* `set(v)` updates value, bumps version, schedules subscribers.

**Tests**

* Get returns initial value.
* Set triggers subscriber execution exactly once.
* Update mutates in place and notifies.
* No subscribers → set is near no-op (version changes only).

**Definition of Done**

* Deterministic ordering & no duplicate notifications in linear chains.

---

## Effects + Batching + Untrack (M3)

**Outcome:** Side-effect computations that react to signal changes.

**Public API**

```go
func CreateEffect($ Scope, fn func())
func Batch($ Scope, fn func())
func Untrack[T any]($ Scope, fn func() T) T
func OnCleanup($ Scope, fn func())
```

**Behavior**

* Effect runs once immediately, then on any dependency change.
* Within `Batch`, multiple sets coalesce into one flush.
* `OnCleanup` runs when effect is re-executed or scope disposed.

**Tests**

* Effect runs once on create.
* Multiple sets in `Batch` → effect runs once.
* `Untrack` prevents dependency capture.

**Definition of Done**

* Topological ordering: upstream effects/memos run before downstream effects.

---

## Memos (M4)

**Outcome:** Derived values with dependency tracking & equality short-circuit.

**Public API**

```go
func CreateMemo[T any]($ Scope, compute func() T, opts ...MemoOption) func() T

type MemoOption = func(*memoCfg)
func WithComparator[T any](eq func(a, b T) bool) MemoOption
```

**Behavior**

* Memo recomputes when any dep version changes.
* If comparator deems equal, memo does **not** notify downstream.

**Tests**

* No recompute when deps unchanged.
* Equality short-circuiting prevents downstream effect runs.
* Memo can depend on other memos and signals (nested graph).

**Definition of Done**

* Stable graph behavior under fan-out and fan-in scenarios.

---

## Context: Provide/Use (M5)

**Outcome:** DI-like mechanism bound to scope.

**Public API**

```go
type ContextKey[T any] struct{ id uintptr }
func Provide[T any]($ Scope, key *ContextKey[T], value T)
func Use[T any]($ Scope, key *ContextKey[T]) T
```

**Tests**

* Provide then Use returns the same value.
* Nested providers shadow parent values.
* Using after dispose panics or returns zero (choose and document).

**Definition of Done**

* Context retrieval is O(1) or near-constant and predictable.

---

## Resources (async keyed fetch) (M6)

**Outcome:** Solid’s `createResource`, Go-style.

**Public API**

```go
type FetchFn[K comparable, T any] func(ctx context.Context, key K) (T, error)

type ResourceHandle[T any] interface {
  Ready() bool
  Value() (T, error) // may block until ready
  Peek() (T, error, bool) // non-blocking
}

func CreateResource[K comparable, T any](
  $ Scope,
  key func() K,
  fetch FetchFn[K, T],
  opts ...ResourceOption,
) ResourceHandle[T]

type ResourceOption = func(*resCfg)
func WithDebounce(d time.Duration) ResourceOption
func WithRetry(n int) ResourceOption
func WithBackoff(min, max time.Duration) ResourceOption
func WithStaleWhileRevalidate(on bool) ResourceOption
```

**Behavior**

* When `key()` changes, cancel current fetch, start new one.
* SWR: immediately serve cached value, refresh in background.
* Debounce before firing fetch.
* Dedupe in-flight by key (optional later).

**Tests**

* Key change triggers cancel → only latest result wins.
* Debounce respected.
* SWR returns previous value while fetching new one.

**Definition of Done**

* Deterministic cancellation + no goroutine leaks (check with `-race` + counters).

---

## Stores (structured state) (M7)

**Outcome:** Convenient nested state with pointer-mutation updates.

**Public API**

```go
func CreateStore[T any]($ Scope, initial T) (
  read func() T,
  set func(mutator func(*T)),
)
```

**Behavior**

* Internally uses a signal; `set` applies a mutator and notifies dependents.
* Optional: field-level comparators (future).

**Tests**

* Mutations propagate like signals.
* Large structs mutate without excessive allocs (compare allocations in bench).

**Definition of Done**

* Same semantics as `CreateSignal` + pointer-update, with nice ergonomics.

---

## Concurrency Options (M8)

**Outcome:** *Opt-in* cross-goroutine safety while keeping single-thread hot path fast.

**Public API (additions)**

```go
type ScopeOption = func(*scopeCfg)
func WithConcurrent() ScopeOption

func RootWith(opts ...ScopeOption, fn func($ Scope)) (dispose func())

// Helpers
func SafeSetter[T any]($ Scope, set func(T)) func(T) // marshals into scope loop
```

**Behavior**

* In concurrent scopes, writes from other goroutines enqueue via an MPSC to the scope’s scheduler.
* Reads are still safe if they go through getters inside the owning goroutine. If you need cross-g access to reads, you either:

  * marshal via a function on the scope loop, or
  * accept eventual consistency with explicit docs.

**Tests**

* Thousands of cross-goroutine sets → consistent order, no data race.
* Dispose while work is enqueued behaves safely.

**Definition of Done**

* `go test -race` consistently clean across stress tests.

---

## Performance Pass + Benchmarks (M9)

**Outcome:** Quantified performance envelope.

**Micro-benchmarks**

* `SignalGet`: aim \~atomic load territory (<10ns).
* `SignalSet(no subs)`: <25ns.
* `SignalSet(10 subs shallow)`: <300ns.
* `MemoStable`: unchanged deps → \~0 extra work (version check only).
* `Resource key flip`: cancel + start in <20µs (excluding network).

**Artifacts**

* `bench/micro/*_test.go` with `testing.B`.
* Track allocations with `b.ReportAllocs()` and target “steady-state zero allocs” for getters.

**Definition of Done**

* Baselines recorded; regressions gated in CI thresholds (even loose ones).

---

## Reliability: Fuzzing, Race, Faults (M10)

**Outcome:** Hardened against edge cases and panics.

**Tasks**

* Table-driven tests for graph shapes (diamonds, cycles).
* **Cycle detection**: detect self-dependency at runtime and panic with a helpful message.
* Fuzz `CreateSignal`/`CreateMemo` interactions (Go fuzzing).
* Inject panics in user callbacks; ensure recover boundary and `OnCleanup` still runs.
* Ensure disposing scopes during flush leaves system consistent.

**Definition of Done**

* Fuzzers run in CI (short time budget).
* Clear error messages; documented failure modes.

---

## Docs, Examples, Versioning (M11)

**Outcome:** New devs can succeed in 5 minutes.

**Docs**

* `README` quickstart with the API you picked.
* `docs/architecture.md` (short): scope, cells, scheduler.
* `docs/perf.md`: results + how to profile.
* `docs/concurrency.md`: what’s safe, patterns, pitfalls.

**Examples**

* `examples/counter` (CLI): signals/memos/effects.
* `examples/http-live-config`: hot-reload config using signals.
* `examples/search-resource`: debounced resource.

**Versioning**

* Tag `v0.x` until API is stable.
* `CHANGELOG.md` with breaking change notes.

**Definition of Done**

* Copy-paste examples compile.
* README includes badge for docs and CI.

---

## Release Checklist (v1.0)

* [ ] Public API frozen: `CreateSignal`, `CreateMemo`, `CreateEffect`, `Batch`, `Untrack`, `Provide/Use`, `CreateResource`, `CreateStore`, `Root`.
* [ ] Deterministic scheduling & disposal documented.
* [ ] `go test ./... -race` clean.
* [ ] Benchmarks checked in with current numbers.
* [ ] Examples compile from a fresh clone.
* [ ] SemVer tag `v1.0.0`.

---

## Appendix: Naming & Public API Reference

**Package**: `github.com/you/signals/pkg/signals`

```go
// Lifetimes
func Root(fn func($ Scope)) (dispose func())
func RootWith(opts ...ScopeOption, fn func($ Scope)) (dispose func())

type Scope interface {
  Batch(fn func())
  Untrack[T any](fn func() T) T
}

// Signals
func CreateSignal[T any]($ Scope, initial T) (
  get func() T,
  set func(T),
  update func(func(*T)),
)

// Memo
type MemoOption = func(*memoCfg)
func CreateMemo[T any]($ Scope, compute func() T, opts ...MemoOption) func() T
func WithComparator[T any](eq func(a, b T) bool) MemoOption

// Effects & Cleanup
func CreateEffect($ Scope, fn func())
func Batch($ Scope, fn func())                  // sugar for $.Batch
func Untrack[T any]($ Scope, fn func() T) T     // sugar for $.Untrack
func OnCleanup($ Scope, fn func())

// Context
type ContextKey[T any] struct{ id uintptr }
func Provide[T any]($ Scope, key *ContextKey[T], value T)
func Use[T any]($ Scope, key *ContextKey[T]) T

// Resource
type FetchFn[K comparable, T any] func(ctx context.Context, key K) (T, error)
type ResourceHandle[T any] interface {
  Ready() bool
  Value() (T, error)
  Peek() (T, error, bool)
}
type ResourceOption = func(*resCfg)
func CreateResource[K comparable, T any](
  $ Scope,
  key func() K,
  fetch FetchFn[K, T],
  opts ...ResourceOption,
) ResourceHandle[T]

func WithDebounce(d time.Duration) ResourceOption
func WithRetry(n int) ResourceOption
func WithBackoff(min, max time.Duration) ResourceOption
func WithStaleWhileRevalidate(on bool) ResourceOption

// Store
func CreateStore[T any]($ Scope, initial T) (
  read func() T,
  set func(mutator func(*T)),
)

// Concurrency
type ScopeOption = func(*scopeCfg)
func WithConcurrent() ScopeOption
func SafeSetter[T any]($ Scope, set func(T)) func(T)
```

---

## Appendix: Testing Tactics

**Unit tests**

* Prefer **table-driven** tests for signals/memos/effects graphs.
* Assert **call order** by appending to a shared slice (protected by scope).

**Property tests (fuzz)**

* Random sequences of `set`/`batch` against known graph shapes; assert invariants (e.g., memo result always equals function of deps).

**Race detector**

* Stress tests with thousands of concurrent `SafeSetter` calls into a concurrent scope.

**Benchmarks**

* Benchmark each primitive in isolation; add downstream subscribers incrementally (1, 5, 10, 50).

**Failure injection**

* Wrap user functions to occasionally panic; assert system recovery and cleanup calls.

---

### Final Notes

* Keep **single-goroutine** default blazing fast. Make concurrency **opt-in**.
* Document **exact semantics** (when effects run, disposal order, how equality short-circuits).
* Keep APIs **tiny**. If you’re unsure about an option, don’t ship it yet—add later with SemVer.

You’ve got this. Build it thin, test it hard, then iterate. The speed you’ll gain from the explicit `Scope` + fine-grained signals will make Go devs do a double-take.
