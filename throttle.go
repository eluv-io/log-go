package log

import (
	"sync"
	"time"

	"github.com/eluv-io/utc-go"
)

type Throttled = ILog

type throttleFactory struct {
	mu    sync.Mutex
	cache map[string]Throttled // throttle key -> Throttled
}

func (f *throttleFactory) get(log *Log, key string, duration ...time.Duration) Throttled {
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
		tl = newThrottledLog(log, dur)
		f.cache[key] = tl
	}
	return tl
}

// newThrottledLog creates a log decorator for throttling log entries.
func newThrottledLog(log *Log, period time.Duration) Throttled {
	return &throttledLog{
		period: period,
		log:    log,
	}
}

// throttledLog is a log decorator that throttles log entries. It logs at most one entry per period, indicating how many
// entries were suppressed in the previous period.
type throttledLog struct {
	log    *Log
	period time.Duration
	mu     sync.Mutex
	count  int
	last   utc.UTC
}

func (l *throttledLog) Throttle(key string, period ...time.Duration) Throttled {
	return l.log.Throttle(key, period...)
}

func (l *throttledLog) IsTrace() bool {
	return l.log.IsTrace()
}

func (l *throttledLog) IsDebug() bool {
	return l.log.IsDebug()
}

func (l *throttledLog) IsInfo() bool {
	return l.log.IsInfo()
}

func (l *throttledLog) IsWarn() bool {
	return l.log.IsWarn()
}

func (l *throttledLog) IsError() bool {
	return l.log.IsError()
}

func (l *throttledLog) IsFatal() bool {
	return l.log.IsFatal()
}

func (l *throttledLog) Trace(msg string, kv ...any) {
	l.throttle(l.log.IsTrace, l.log.Trace, msg, kv...)
}

func (l *throttledLog) Debug(msg string, kv ...any) {
	l.throttle(l.log.IsDebug, l.log.Debug, msg, kv...)
}

func (l *throttledLog) Info(msg string, kv ...any) {
	l.throttle(l.log.IsInfo, l.log.Info, msg, kv...)
}

func (l *throttledLog) Warn(msg string, kv ...any) {
	l.throttle(l.log.IsWarn, l.log.Warn, msg, kv...)
}

func (l *throttledLog) Error(msg string, kv ...any) {
	l.throttle(l.log.IsError, l.log.Error, msg, kv...)
}

func (l *throttledLog) Fatal(msg string, kv ...any) {
	l.log.Fatal(msg, kv...)
}

func (l *throttledLog) throttle(isFn func() bool, logFn func(msg string, kv ...any), msg string, kv ...any) {
	if !isFn() {
		return
	}

	skip := false
	l.mu.Lock()
	if l.last.IsZero() {
		l.last = utc.Now()
	} else if utc.Since(l.last) >= l.period {
		kv = append(kv, "suppressed", l.count, "throttle_period", l.period)
		l.count = 0
		l.last = utc.Now()
	} else {
		l.count++
		skip = true
	}
	l.mu.Unlock()
	if skip {
		return
	}
	logFn(msg, kv...)
}
