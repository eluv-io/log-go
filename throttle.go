package log

import (
	"sync"
	"time"

	apex "github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/utc-go"
)

type Throttled interface {
	Trace(msg string, kv ...interface{})
	Debug(msg string, kv ...interface{})
	Info(msg string, kv ...interface{})
	Warn(msg string, kv ...interface{})
	Error(msg string, kv ...interface{})
	Fatal(msg string, kv ...interface{})
}

type throttleFactory struct {
	mu    sync.Mutex
	cache map[string]Throttled // throttle key -> Throttled
}

func (f *throttleFactory) get(logger *apex.Logger, key string, duration ...time.Duration) Throttled {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cache == nil {
		f.cache = make(map[string]Throttled)
	}
	tl, ok := f.cache[key]
	if !ok {
		dur := 5 * time.Second
		if len(duration) > 0 {
			dur = duration[0]
		}
		tl = newThrottledLog(logger, dur)
		f.cache[key] = tl
	}
	return tl
}

// newThrottledLog creates a log decorator for throttling similar log entries.
func newThrottledLog(logger *apex.Logger, period time.Duration) apex.Interface {
	return &throttledLog{
		period: period,
		Logger: logger,
	}
}

// newThrottledLog is a log decorator that throttles similar log entries. Similarity is explicitly signalled by the
// application by specifying a key/value pair in the log statement, where the key corresponds to the configured
// throttling key and the value matches that of "similar" statements.
type throttledLog struct {
	period time.Duration
	*apex.Logger
	mu    sync.Mutex
	count int
	last  utc.UTC
}

func (f *throttledLog) Trace(msg string, kv ...any) {
	f.throttle(f.Logger.Trace, msg, kv...)
}

func (f *throttledLog) Debug(msg string, kv ...any) {
	f.throttle(f.Logger.Debug, msg, kv...)
}

func (f *throttledLog) Info(msg string, kv ...any) {
	f.throttle(f.Logger.Info, msg, kv...)
}

func (f *throttledLog) Warn(msg string, kv ...any) {
	f.throttle(f.Logger.Warn, msg, kv...)
}

func (f *throttledLog) Error(msg string, kv ...any) {
	f.throttle(f.Logger.Error, msg, kv...)
}

func (f *throttledLog) Fatal(msg string, kv ...any) {
	f.throttle(f.Logger.Fatal, msg, kv...)
}

func (f *throttledLog) throttle(logFn func(msg string, kv ...any), msg string, kv ...any) {
	skip := false
	f.mu.Lock()
	if f.last.IsZero() {
		f.last = utc.Now()
	} else if utc.Since(f.last) >= f.period {
		kv = append(kv, "suppressed", f.count, "throttle_period", f.period)
		f.count = 0
		f.last = utc.Now()
	} else {
		f.count++
		skip = true
	}
	f.mu.Unlock()
	if skip {
		return
	}
	logFn(msg, kv...)
}
