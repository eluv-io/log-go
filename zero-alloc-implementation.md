# Zero-Allocation Logging — Phase 1 Implementation

## Overview

Phase 1 adds a zero-allocation fast path to `log-go` via six new methods on `*Log`. The entire
existing API is unchanged; the fast path is strictly additive.

---

## New API

Six methods are added to `*Log` (not to `ILog`, to avoid breaking existing implementors):

```go
func (l *Log) TraceEvent() *zerolog.Event
func (l *Log) DebugEvent() *zerolog.Event
func (l *Log) InfoEvent()  *zerolog.Event
func (l *Log) WarnEvent()  *zerolog.Event
func (l *Log) ErrorEvent() *zerolog.Event
func (l *Log) FatalEvent() *zerolog.Event
```

Each method returns a `*zerolog.Event`. Fields are appended with typed builder methods
(`Str`, `Int`, `Int64`, `Bool`, `Err`, …). The entry is committed by calling `Msg()` or `Send()`.

```go
// existing API — unchanged
log.Info("request completed", "method", r.Method, "status", status, "duration_ms", dur.Milliseconds())

// fast path — zero allocations
log.InfoEvent().
    Str("method", r.Method).
    Int("status", status).
    Int64("duration_ms", dur.Milliseconds()).
    Msg("request completed")
```

When a level is disabled, `*Event()` returns a no-op `*zerolog.Event` — all chained calls are
elided. No allocation, no level-guard boilerplate needed for simple field expressions:

```go
// no guard needed — chained calls are free when debug is disabled
log.DebugEvent().Str("method", r.Method).Int("status", code).Msg("handled")

// guard still needed when the argument itself is expensive to compute
if log.IsDebug() {
    log.DebugEvent().Str("dump", expensive()).Msg("state")
}
```

---

## Benchmark Results

Measured on Apple M4 Max with a discard handler and 10 fields:

| Path                            | ns/op | B/op  | allocs/op |
|---------------------------------|-------|-------|-----------|
| existing vararg, level disabled | 56    | 320   | 1         |
| existing vararg, level enabled  | 862   | 1,794 | 28        |
| `DebugEvent`, level disabled    | 9     | 0     | **0**     |
| `InfoEvent`, level enabled      | 198   | 0     | **0**     |

The fast path is **4× faster** on the enabled path and **6× faster** on the disabled path, with
zero heap allocations in both cases.

---

## Files Changed

### New: `fast_log.go`

Contains:

- The six `*Event()` methods on `*Log`
- `apexToZerologLevel(apex.Level) zerolog.Level` — maps apex levels to zerolog levels
- `newZerologLogger(w, level, handlerType, loggerName)` — constructs a `zerolog.Logger` whose
  output format approximates the configured apex handler type (see Output Format below)
- `zlFormatLevel` / `zlFormatMessage` — ConsoleWriter formatters matching the apex text format

### Modified: `logger.go`

Two fields added to the `logger` struct:

```go
zlWriter io.Writer      // underlying writer shared with the zerolog logger
zl       zerolog.Logger // zero-allocation fast-path logger
```

`logger.copy()` propagates both fields so that level changes and reconfigurations apply
correctly to the zerolog logger via the `modFns` mechanism.

### Modified: `log_impl.go`

`newLog()` now creates a zerolog logger alongside the apex logger. A new helper
`zerologWriter()` determines the correct `io.Writer`:

- `discard` / `memory` handlers → `io.Discard`
- Handler reused from parent (same type + same file config) → parent's `zlWriter`, ensuring
  both loggers share the same output destination
- New handler → the newly created writer (`os.Stdout` or lumberjack file)

### Modified: `log.go`

`setLogLevel()` now also updates the zerolog level on the copied logger:

```go
logCopy.zl = logCopy.zl.Level(apexToZerologLevel(level))
```

This keeps both loggers in sync whenever `SetLevel`, `SetDebug`, etc. are called.

### Modified: `bench_test.go`

Added `BenchmarkFastLog` with two sub-benchmarks: an enabled `InfoEvent` path and a disabled
`DebugEvent` path, both with 10 fields.

---

## Output Format

The zerolog fast path writes to the same `io.Writer` as the apex handler. The format is chosen
per handler type:

| Handler             | zerolog format                                                                              |
|---------------------|---------------------------------------------------------------------------------------------|
| `json`              | Native zerolog JSON                                                                         |
| `text`, `raw`       | `zerolog.ConsoleWriter` (no colour), level labels and message padding approximate apex text |
| `console`           | `zerolog.ConsoleWriter` (with colour)                                                       |
| `discard`, `memory` | Discarded (`io.Discard`)                                                                    |

For the text and raw handlers the ConsoleWriter is configured with:

- `TimeFormat: "2006-01-02T15:04:05.000Z07:00"` — matching apex's millisecond UTC timestamps
- Custom `FormatLevel` producing `INFO `, `WARN `, `DEBUG`, `ERROR`, `FATAL`, `TRACE`
- Custom `FormatMessage` applying `%-25s` padding, matching the apex text handler layout

For JSON the formats are structurally compatible (both emit `{"level":…,"time":…,"message":…}`),
though field ordering and the exact timestamp precision may differ until Phase 2.

---

## What Was Not Changed

- `ILog` interface — no new methods; all existing implementors compile unchanged
- `noop.go` — the no-op logger is unaffected
- `throttle.go` — the throttle decorator is unaffected
- `log_config.go` — `Config`, `LumberjackConfig`, `Stdout` unchanged
- `Log.Rotate()` — lumberjack integration unchanged
- All existing handlers (`text`, `raw`, `console`) — unchanged

---

## Next Step: Phase 2

Phase 2 replaces the apexlog backend with zerolog entirely, eliminating the format divergence
and making the existing `Trace/…/Fatal(msg, kv ...interface{})` path faster as well. See
`zero-alloc-design.md` for the full plan.
