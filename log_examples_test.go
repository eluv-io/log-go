package log_test

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/eluv-io/log-go"
	"github.com/eluv-io/utc-go"
)

// ExampleLog_InfoEvent demonstrates the zero-allocation fast-path API using zerolog event builders.
// The logger is configured with the json handler so the output format is visible.
//
// Note: zerolog's field order differs slightly from apex's — level and context fields (logger,
// time) bookend the event-specific fields. This is a Phase 1 characteristic; Phase 2 will unify
// the format by replacing the apex backend entirely.
func ExampleLog_InfoEvent() {
	defer utc.MockNowFn(func() utc.UTC { return utc.UnixMilli(0) })()

	logger := log.New(&log.Config{Handler: "json"})

	logger.InfoEvent().
		Str("api", "zerolog").
		Str("file", "photo.jpg").
		Int("size", 1024).
		Msg("upload complete")

	logger.WarnEvent().
		Str("api", "zerolog").
		Str("file", "photo.jpg").
		Err(errors.New("disk full")).
		Msg("upload failed")

	// Output:
	// {"level":"info","logger":"/","api":"zerolog","file":"photo.jpg","size":1024,"time":"1970-01-01T00:00:00.000Z","message":"upload complete"}
	// {"level":"warn","logger":"/","api":"zerolog","file":"photo.jpg","error":"disk full","time":"1970-01-01T00:00:00.000Z","message":"upload failed"}
}

// ExampleLog_mixed demonstrates the zerolog fast-path and the classic vararg API writing to the
// same destination within a single logger. Both paths produce one line per call.
//
// The main visible difference between the two paths (Phase 1) is field order: apex preserves
// insertion order while zerolog's ConsoleWriter sorts fields alphabetically.
func ExampleLog_mixed() {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger := log.New(&log.Config{Handler: "text"})

	// classic vararg API — apex text handler, insertion-order fields
	logger.Info("upload started", "api", "regular", "file", "photo.jpg")

	// zero-allocation fast path — zerolog ConsoleWriter, alphabetically sorted fields
	logger.InfoEvent().
		Str("api", "zerolog").
		Str("file", "photo.jpg").
		Int("size", 1024).
		Msg("upload complete")

	logger.Warn("quota warning", "api", "regular", "used_pct", 92)

	logger.WarnEvent().
		Str("api", "zerolog").
		Int("used_pct", 92).
		Msg("quota warning")

	// Output:
	// 1970-01-01T00:00:00.000Z INFO  upload started            logger=/ api=regular file=photo.jpg
	// 1970-01-01T00:00:00.000Z INFO  upload complete           api=zerolog file=photo.jpg logger=/ size=1024
	// 1970-01-01T00:00:00.000Z WARN  quota warning             logger=/ api=regular used_pct=92
	// 1970-01-01T00:00:00.000Z WARN  quota warning             api=zerolog logger=/ used_pct=92
}

// ExampleLog_jsonValue demonstrates logging a value's JSON representation using Interface. The
// value is marshaled via json.Marshal and embedded as a JSON object in the log record.
//
// Note: Interface takes interface{}, so passing a concrete value boxes it onto the heap and
// incurs at least one allocation — the zero-allocation guarantee does not hold for this method.
// For hot paths, prefer RawJSON(key, preSerializedBytes) which appends bytes directly into the
// event buffer without boxing or marshaling.
func ExampleLog_jsonValue() {
	defer utc.MockNowFn(func() utc.UTC { return utc.UnixMilli(0) })()

	type Dimensions struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	logger := log.New(&log.Config{Handler: "json"})

	img := Dimensions{Width: 1920, Height: 1080}

	// Interface marshals the value via json.Marshal — convenient but boxes the argument onto the heap.
	logger.InfoEvent().
		Interface("dimensions", img).
		Msg("image processed")

	// RawJSON embeds pre-serialized bytes directly — zero allocations on the log call itself.
	b, _ := json.Marshal(img)
	logger.InfoEvent().
		RawJSON("dimensions", b).
		Msg("image processed")

	// Output:
	// {"level":"info","logger":"/","dimensions":{"width":1920,"height":1080},"time":"1970-01-01T00:00:00.000Z","message":"image processed"}
	// {"level":"info","logger":"/","dimensions":{"width":1920,"height":1080},"time":"1970-01-01T00:00:00.000Z","message":"image processed"}
}

func ExampleLog_throttle() {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	fls := false
	logger := log.New(
		&log.Config{
			Handler:     "text",
			GoRoutineID: &fls,
		})

	for i := 1; i < 25; i++ {
		err := errors.New("connect error")
		logger.Throttle("connect", 100*time.Millisecond).Warn("failed to connect", err, "attempt", i)
		now = now.Add(10 * time.Millisecond)
	}

	// Output:
	//
	// 1970-01-01T00:00:00.000Z WARN  failed to connect         logger=/ attempt=1 error=connect error
	// 1970-01-01T00:00:00.100Z WARN  failed to connect         logger=/ attempt=11 suppressed=9 throttle_period=100ms error=connect error
	// 1970-01-01T00:00:00.200Z WARN  failed to connect         logger=/ attempt=21 suppressed=9 throttle_period=100ms error=connect error
}
