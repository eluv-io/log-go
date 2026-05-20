# Zero-Allocation Logging API — Design Proposal

## Background

`log-go` exposes a flexible, easy-to-use structured logging API:

```go
log.Info("user logged in", "user_id", uid, "ip", ip, "duration_ms", dur.Milliseconds())
```

The variadic `...interface{}` signature makes this ergonomic, but every call pays an allocation tax that
cannot be avoided at the language level. For hot paths — request handlers, tight loops, high-throughput
pipelines — this overhead is meaningful.

This document analyses the root causes, evaluates backend replacement options, and proposes a migration
path to a zero-allocation API that preserves full backward compatibility.

---

## Root Cause of Current Allocations

Every call to a logging method triggers at least three layers of allocation:

### 1. Variadic slice allocation

```go
func (l *Log) Info(msg string, kv ...interface{})
```

The compiler must allocate a `[]interface{}` on the heap for each call site. Even when the log level is
disabled and the slice is never read, the allocation still occurs before the level check.

### 2. Interface boxing

Each concrete value passed as a key or value is boxed into `interface{}`. Primitive types (int, bool,
float64) that do not fit in a pointer word cause additional small heap allocations.

### 3. The apex layer

The underlying `github.com/eluv-io/apexlog-go` backend allocates an `apex.Entry` and an `apex.Fields`
slice (a `[]Field` of `{Name string, Value interface{}}` structs) for every log call that reaches a
handler. This is unavoidable given the apex data model.

### Benchmark evidence

From `bench_test.go` (discard handler, so I/O is excluded):

| Scenario | ns/op | B/op | allocs/op |
|---|---|---|---|
| File handler, typical call | ~17,048 | 3,767 | 51 |
| Discard handler | ~4,302 | 1,849 | 31 |
| Disabled level (debug on info config) | ~131 | 320 | 1 |

Even a disabled log level costs **1 allocation** (the variadic slice). An enabled call through a
discard handler costs **31 allocations**. No amount of tuning within the current `...interface{}`
contract can eliminate these.

---

## Backend Replacement Options

Achieving genuine zero allocations requires a fundamentally different API contract. Two candidates are
worth evaluating: **zerolog** and the standard library **slog**.

### zerolog (`github.com/rs/zerolog`)

zerolog uses a **builder chain** pattern backed by a `sync.Pool` of `[]byte` event buffers:

```go
log.Info().Str("user_id", uid).Int("count", n).Msg("logged in")
```

How it achieves zero allocations:
- `Event` objects are pooled via `sync.Pool` — no allocation to start a log entry
- Each typed method (`Str`, `Int`, `Bool`, `Err`, …) appends bytes directly to the buffer — no boxing,
  no intermediate structs
- `Msg()` flushes the buffer to the writer and returns the `Event` to the pool

zerolog's own benchmarks consistently show **0 allocs/op** for the builder path. It is widely adopted,
has 10k+ stars, and is battle-tested in high-throughput production systems.

The `zerolog.Logger` is a value type (not a pointer) and is safe to copy and embed.

### slog (standard library, Go 1.21+)

slog provides `LogAttrs` as its low-allocation path:

```go
slog.LogAttrs(ctx, slog.LevelInfo, "logged in",
    slog.String("user_id", uid), slog.Int("count", n))
```

`slog.Value` uses a tagged union to store common types (bool, int64, float64, string, time.Time,
Duration) inline without boxing. This is a meaningful improvement over `...interface{}`.

However, slog does **not** achieve 0 allocs/op:
- The `[]slog.Attr` slice passed to the handler is typically heap-allocated
- Custom `slog.Handler` implementations almost always allocate
- slog's own documentation does not claim zero allocations; benchmarks show 1–3 allocs/op in typical
  usage

slog's main advantage is that it is part of the standard library — no extra dependency. But for a
logging library whose explicit goal is zero allocations, slog does not clear the bar.

### Verdict

| | zerolog | slog |
|---|---|---|
| Zero allocs/op (builder path) | Yes | No (1–3 typical) |
| Typed field methods | Yes | Yes |
| Standard library | No | Yes |
| Ecosystem / adoption | Large, mature | Growing |
| `io.Writer`-based output | Yes | Yes (via handler) |
| Value-type logger (copyable) | Yes | Yes |

**zerolog is the right choice** for a genuine zero-allocation commitment. slog reduces allocations but
cannot eliminate them, and introducing it as a backend would not deliver on the stated goal.

---

## Recommended Approach: Two-Phase Migration

The migration is split into two phases so that Phase 1 ships quickly with no risk to existing consumers,
and Phase 2 delivers the complete architectural simplification as a follow-up.

### Phase 1 — Additive zero-allocation fast path

Keep the entire existing API unchanged. Add zerolog as a second backend alongside apexlog. Both share
the same underlying `io.Writer` so output goes to the same destination.

Each `Log` struct gains a `zerolog.Logger` field, updated atomically alongside the existing apex logger
whenever the configuration changes. New entry-point methods are added to `*Log`:

```go
func (l *Log) TraceEvent() *zerolog.Event
func (l *Log) DebugEvent() *zerolog.Event
func (l *Log) InfoEvent()  *zerolog.Event
func (l *Log) WarnEvent()  *zerolog.Event
func (l *Log) ErrorEvent() *zerolog.Event
func (l *Log) FatalEvent() *zerolog.Event
```

These methods are added **only to `*Log`**, not to the `ILog` interface. Adding them to `ILog` would be
a breaking change for all implementors (the noop logger, throttle decorator, and any consumer-defined
mocks). The vast majority of callers already hold a `*Log`, so this is not a practical limitation.

#### Usage — before and after

```go
// Before: allocates slice + boxes 6 values + apex Entry/Fields
log.Info("request completed",
    "method", r.Method,
    "status", status,
    "duration_ms", dur.Milliseconds())

// After: zero allocations
log.InfoEvent().
    Str("method", r.Method).
    Int("status", status).
    Int64("duration_ms", dur.Milliseconds()).
    Msg("request completed")
```

#### Disabled levels are free

zerolog's `DebugEvent()` returns a no-op `*zerolog.Event` when the debug level is disabled. Chained
method calls on a no-op event compile to nothing significant. However, if the argument itself is
expensive to compute, the caller must still guard it:

```go
// compute() is called regardless — guard needed
if log.IsDebug() {
    log.DebugEvent().Str("dump", expensive()).Msg("state")
}

// literals and simple field reads — no guard needed
log.DebugEvent().Str("method", r.Method).Int("status", code).Msg("handled")
```

This is identical behaviour to the existing API, where `IsDebug()` guards are already recommended for
expensive arguments.

#### Output format

For JSON output the formats are already compatible: both apex and zerolog emit standard
`{"level":"info","message":"...","key":"value"}` JSON. For text and console output, zerolog's
`ConsoleWriter` would need to be configured to match the existing apex text format, or the format
difference is accepted and documented as a characteristic of the fast path. This is a reasonable
trade-off — callers choosing the fast path are optimising for throughput, not log aesthetics.

### Phase 2 — Replace apexlog with zerolog entirely

Once Phase 1 has been in production and the zerolog integration is validated, remove apexlog:

- Each `Log` holds only a `zerolog.Logger`; the apex dependency is dropped
- The slow `Trace/Debug/…/Fatal(msg string, kv ...interface{})` methods are reimplemented on top of
  zerolog's `Dict()` / `Fields()` builder internally — they still allocate due to `...interface{}`, but
  the apex Entry/Fields allocation layer disappears, roughly halving per-call allocations on the slow
  path
- All existing handlers (`text`, `raw`, `console`) are rewritten as `zerolog.LevelWriter`
  implementations — the core format logic is approximately 100 lines each
- The `ILog` interface, the `Get()` / `Root()` / `SetDefault()` functions, the `Config` struct, the
  throttle decorator, and the noop logger all remain unchanged
- All existing consumers compile and behave identically

Phase 2 eliminates the format divergence between fast and slow paths, simplifies the codebase by
removing a dependency, and makes the slow path faster as a bonus.

---

## Implementation Sketch (Phase 1)

```go
// Log struct — add zerolog alongside existing apex logger
type Log struct {
    lw atomic.Pointer[logger]         // existing
    zl atomic.Pointer[zerolog.Logger] // new: fast path
}

// InfoEvent returns a zero-allocation zerolog event at Info level.
// Call Msg() or Send() on the returned event to emit the log entry.
func (l *Log) InfoEvent() *zerolog.Event {
    if zl := l.zl.Load(); zl != nil {
        return zl.Info()
    }
    return zerolog.Nop().Info()
}

// DebugEvent returns a zero-allocation zerolog event at Debug level.
// Returns a no-op event when the debug level is disabled.
func (l *Log) DebugEvent() *zerolog.Event {
    if zl := l.zl.Load(); zl != nil {
        return zl.Debug()
    }
    return zerolog.Nop().Debug()
}

// … WarnEvent, ErrorEvent, TraceEvent, FatalEvent follow the same pattern
```

When a `Log` is (re)configured, the zerolog logger is constructed from the same `io.Writer` that the
apex handler uses:

```go
func newZerologLogger(w io.Writer, level zerolog.Level) zerolog.Logger {
    return zerolog.New(w).Level(level).With().Timestamp().Logger()
}
```

The zerolog level is kept in sync with the apex level in the existing `reconfigure` path so both loggers
always agree on what is enabled.

---

## Log Rotation: lumberjack and Alternatives

### zerolog and log rotation

zerolog has no native log rotation. It writes to any `io.Writer` and delegates rotation entirely to
that writer — the right design for a logging library, since rotation is an I/O concern, not a
formatting one.

This means the entire rotation layer — `LumberjackConfig`, `NewLumberjackLogger()`, `Log.Rotate()` —
survives both Phase 1 and Phase 2 of the migration completely unchanged. lumberjack integrates with
zerolog identically to how it integrates with apex today: both accept an `io.Writer`.

### Is lumberjack worth replacing?

The current dependency is `gopkg.in/natefinch/lumberjack.v2`. It is in maintenance mode but
functionally complete: the code is small (~400 lines), the size/age/compress feature set covers
virtually all use cases, and it is the de facto standard used by every major Go logging library
including zap. The quiet upstream is not a practical concern.

The main alternatives, and when to prefer them:

| Option | Rotation trigger | Pros | Cons |
|---|---|---|---|
| **lumberjack** (current) | File size / age, in-process | Self-contained, zero ops setup, exposes `Rotate()` | Maintenance-mode upstream; edge cases on Windows |
| **OS `logrotate` + SIGHUP** | Cron / systemd, out-of-process | Production-grade standard on Linux; no in-process state | Requires ops setup; app must reopen file on signal |
| **`lestrrat-go/file-rotatelogs`** | Time-based (hourly/daily) | Symlink to current file, clean dated naming | Less maintained; no size-based trigger |

For a library that embeds logging in another service, lumberjack remains the right default. If the
consuming service runs under systemd or in a container with a log aggregator (Loki, Fluentd, Vector,
etc.), the best practice is to write to stdout/stderr and let the platform handle rotation — already
supported via the `Stdout` config variable in this library.

**Conclusion:** no changes needed on the rotation front. lumberjack stays as-is across both migration
phases.

---

## Migration Summary

| Phase | What changes | Risk | Backward compatible |
|---|---|---|---|
| Phase 1: additive fast path | Add zerolog dep; new `*Event()` methods on `*Log` | Low | Yes — existing API untouched |
| Phase 2: replace apexlog | Remove apex dep; rewrite handlers as zerolog writers | Medium | Yes — `ILog` interface unchanged |
| slog backend (not recommended) | Replace apex with slog | Low | Yes | 

Phase 1 can be shipped immediately and gives callers an opt-in zero-allocation path. Phase 2 is the
right long-term destination once the zerolog integration is validated in real workloads.
