package log_test

import (
	"errors"
	"time"

	"github.com/eluv-io/log-go"
	"github.com/eluv-io/utc-go"
)

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
	// 1970-01-01T00:00:00.000Z WARN  failed to connect         attempt=1 error=connect error
	// 1970-01-01T00:00:00.100Z WARN  failed to connect         attempt=11 suppressed=9 throttle_period=100ms error=connect error
	// 1970-01-01T00:00:00.200Z WARN  failed to connect         attempt=21 suppressed=9 throttle_period=100ms error=connect error
}
