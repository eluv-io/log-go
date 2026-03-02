package log_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/apexlog-go/handlers/memory"

	"github.com/eluv-io/log-go"
	"github.com/eluv-io/utc-go"
)

func newThrottleTestLogger(level string) (*log.Log, *memory.Handler) {
	l := log.New(&log.Config{
		Handler: "memory",
		Level:   level,
	})
	return l, l.Handler().(*memory.Handler)
}

// TestThrottleFactory_SameKeyReturnsSameInstance verifies that calling Throttle with the same key always returns the
// same throttled logger instance.
func TestThrottleFactory_SameKeyReturnsSameInstance(t *testing.T) {
	logger, _ := newThrottleTestLogger("info")

	t1 := logger.Throttle("key1")
	t2 := logger.Throttle("key1")

	assert.True(t, t1 == t2, "same key must return the same Throttled instance")
}

// TestThrottleFactory_DifferentKeysAreIndependent verifies that different throttle keys produce separate instances that
// track state independently.
func TestThrottleFactory_DifferentKeysAreIndependent(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("info")

	t1 := logger.Throttle("key1", 100*time.Millisecond)
	t2 := logger.Throttle("key2", 100*time.Millisecond)

	assert.False(t, t1 == t2, "different keys must return different instances")

	// Each key throttles independently: each should emit its first message
	t1.Info("from key1")
	t1.Info("from key1 suppressed")
	t2.Info("from key2")
	t2.Info("from key2 suppressed")

	require.Len(t, handler.Entries, 2)
	assert.Equal(t, "from key1", handler.Entries[0].Message)
	assert.Equal(t, "from key2", handler.Entries[1].Message)
}

// TestThrottle_DefaultPeriod verifies that the default throttle period is 5 seconds.
func TestThrottle_DefaultPeriod(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("info")
	throttled := logger.Throttle("key") // no explicit period → default 5s

	// First message is logged
	throttled.Info("msg")
	require.Len(t, handler.Entries, 1)

	// Still within default 5s period — suppressed
	now = now.Add(4 * time.Second)
	throttled.Info("msg")
	require.Len(t, handler.Entries, 1)

	// Past the 5s default period — logged again
	now = now.Add(2 * time.Second)
	throttled.Info("msg")
	require.Len(t, handler.Entries, 2)
}

// TestThrottle_CustomPeriod verifies that a custom throttle period is honoured.
func TestThrottle_CustomPeriod(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("info")
	throttled := logger.Throttle("key", 200*time.Millisecond)

	throttled.Info("msg")
	require.Len(t, handler.Entries, 1)

	// Within the custom period — suppressed
	now = now.Add(100 * time.Millisecond)
	throttled.Info("msg")
	require.Len(t, handler.Entries, 1)

	// Past the custom period — logged again
	now = now.Add(200 * time.Millisecond)
	throttled.Info("msg")
	require.Len(t, handler.Entries, 2)
}

// TestThrottle_SuppressedCountAndPeriodField verifies that the "suppressed" count and "throttle_period" fields are
// correctly appended to the first message emitted after a throttle period expires.
func TestThrottle_SuppressedCountAndPeriodField(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("info")
	period := 100 * time.Millisecond
	throttled := logger.Throttle("key", period)

	// First call: logged without suppression metadata
	throttled.Info("msg", "attempt", 1)
	require.Len(t, handler.Entries, 1)
	assert.Nil(t, handler.Entries[0].Fields.Get("suppressed"), "first entry must not contain suppressed field")

	// 9 more calls within the period: all suppressed
	for i := 2; i <= 10; i++ {
		throttled.Info("msg", "attempt", i)
	}
	require.Len(t, handler.Entries, 1)

	// Advance past period; next call must include suppressed=9 and throttle_period
	now = now.Add(200 * time.Millisecond)
	throttled.Info("msg", "attempt", 11)
	require.Len(t, handler.Entries, 2)

	entry := handler.Entries[1]
	assert.Equal(t, 9, entry.Fields.Get("suppressed"))
	assert.Equal(t, period, entry.Fields.Get("throttle_period"))
	assert.Equal(t, 11, entry.Fields.Get("attempt"))
}

// TestThrottle_SuppressedCountResetsEachPeriod verifies that the suppressed counter resets at the start of each new
// period.
func TestThrottle_SuppressedCountResetsEachPeriod(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("info")
	period := 100 * time.Millisecond
	throttled := logger.Throttle("key", period)

	// First period: log once then suppress 5 times
	throttled.Info("msg")
	for i := 0; i < 5; i++ {
		throttled.Info("suppressed")
	}

	// Start of second period: should report suppressed=5 from first period
	now = now.Add(200 * time.Millisecond)
	throttled.Info("after first period")
	require.Len(t, handler.Entries, 2)
	assert.Equal(t, 5, handler.Entries[1].Fields.Get("suppressed"))

	// Start of third period: no suppressions happened in second period → suppressed=0
	now = now.Add(200 * time.Millisecond)
	throttled.Info("after second period")
	require.Len(t, handler.Entries, 3)
	assert.Equal(t, 0, handler.Entries[2].Fields.Get("suppressed"))
}

// TestThrottle_LevelChecks verifies that the IsXxx level check methods on throttledLog correctly delegate to the
// underlying logger.
func TestThrottle_LevelChecks(t *testing.T) {
	for _, tc := range []struct {
		level   string
		isTrace bool
		isDebug bool
		isInfo  bool
		isWarn  bool
		isError bool
		isFatal bool
	}{
		{"trace", true, true, true, true, true, true},
		{"debug", false, true, true, true, true, true},
		{"info", false, false, true, true, true, true},
		{"warn", false, false, false, true, true, true},
		{"error", false, false, false, false, true, true},
	} {
		t.Run(tc.level, func(t *testing.T) {
			logger := log.New(&log.Config{Handler: "memory", Level: tc.level})
			tl := logger.Throttle("key")

			assert.Equal(t, tc.isTrace, tl.IsTrace(), "IsTrace")
			assert.Equal(t, tc.isDebug, tl.IsDebug(), "IsDebug")
			assert.Equal(t, tc.isInfo, tl.IsInfo(), "IsInfo")
			assert.Equal(t, tc.isWarn, tl.IsWarn(), "IsWarn")
			assert.Equal(t, tc.isError, tl.IsError(), "IsError")
			assert.Equal(t, tc.isFatal, tl.IsFatal(), "IsFatal")
		})
	}
}

// TestThrottle_BelowLevelDoesNotAffectState verifies that log calls below the configured log level are silently dropped
// and do not affect the throttle state (i.e. they do not consume the first-message slot or increment the suppressed
// count).
func TestThrottle_BelowLevelDoesNotAffectState(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("warn") // only warn and above
	throttled := logger.Throttle("key", 100*time.Millisecond)

	// Debug and Info calls are below level and must not affect throttle state
	for i := 0; i < 10; i++ {
		throttled.Debug("below level")
		throttled.Info("below level")
	}
	require.Len(t, handler.Entries, 0)

	// First Warn is logged and starts the throttle window
	throttled.Warn("at warn level")
	require.Len(t, handler.Entries, 1)

	// Below-level calls after the timer is set still don't affect state
	throttled.Debug("below level")
	throttled.Info("below level")

	// Second Warn within the period is throttled
	throttled.Warn("still throttled")
	require.Len(t, handler.Entries, 1)

	// After the period, the next Warn is logged with suppressed=1
	now = now.Add(200 * time.Millisecond)
	throttled.Warn("after period")
	require.Len(t, handler.Entries, 2)
	assert.Equal(t, 1, handler.Entries[1].Fields.Get("suppressed"))
}

// TestThrottle_NestedThrottle verifies that calling Throttle on a throttledLog correctly delegates to the underlying
// logger's throttle factory.
func TestThrottle_NestedThrottle(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("info")

	t1 := logger.Throttle("key1", 100*time.Millisecond)

	// Throttle obtained from a throttledLog should use a separate key
	t2 := t1.Throttle("key2", 100*time.Millisecond)
	require.NotNil(t, t2)

	// t1 and t2 are independent: each logs its first message
	t1.Info("from t1")
	t2.Info("from t2")
	require.Len(t, handler.Entries, 2)

	// Suppress subsequent messages within the period
	t1.Info("t1 suppressed")
	t2.Info("t2 suppressed")
	require.Len(t, handler.Entries, 2)
}

// TestThrottle_FirstMessageAlwaysLogged verifies that the very first call on a freshly created throttled logger always
// logs (last time is zero).
func TestThrottle_FirstMessageAlwaysLogged(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	logger, handler := newThrottleTestLogger("info")

	throttled := logger.Throttle("fresh-key", 1*time.Hour)
	throttled.Info("first ever message")
	require.Len(t, handler.Entries, 1)
}

// TestThrottle_Concurrent verifies that concurrent calls to the same throttled logger do not cause data races. Run with
// -race to validate.
func TestThrottle_Concurrent(t *testing.T) {
	logger, _ := newThrottleTestLogger("info")
	throttled := logger.Throttle("key", 100*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			throttled.Info("concurrent", "i", i)
		}(i)
	}
	wg.Wait()
}

// TestThrottleFactory_ConcurrentGet verifies that concurrent calls to Throttle with the same key always return the same
// instance and do not race. Run with -race.
func TestThrottleFactory_ConcurrentGet(t *testing.T) {
	logger, _ := newThrottleTestLogger("info")

	results := make([]log.Throttled, 100)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = logger.Throttle("same-key")
		}(i)
	}
	wg.Wait()

	first := results[0]
	for _, r := range results[1:] {
		assert.True(t, first == r, "all concurrent Throttle calls must return the same instance")
	}
}

// TestThrottle_Reconfig tests that reconfiguration of the underlying logger is propagated to the throttled log.
func TestThrottle_Reconfig(t *testing.T) {
	now := utc.UnixMilli(0)
	defer utc.MockNowFn(func() utc.UTC { return now })()

	log.SetDefault(&log.Config{
		Handler: "memory",
		Level:   "info",
	})

	logger := log.Get("/test")
	handler := logger.Handler().(*memory.Handler)

	throttled := logger.Throttle("key")

	// First message is ignored
	throttled.Debug("debug ignored")
	require.Len(t, handler.Entries, 0)

	log.SetDefault(&log.Config{
		Handler: "memory",
		Level:   "debug",
	})
	handler = logger.Handler().(*memory.Handler) // need to retrieve the handler again after reconfig

	// First message is logged
	throttled.Debug("debug")
	require.Len(t, handler.Entries, 1)
	require.Equal(t, "debug", handler.Entries[0].Message)

}
