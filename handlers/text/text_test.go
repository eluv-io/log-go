package text_test

import (
	"github.com/eluv-io/log-go"
	"github.com/eluv-io/utc-go"
)

func ExampleHandler() {
	defer utc.MockNow(utc.UnixMilli(0))()

	fls := false
	lg := log.New(&log.Config{
		Level:       "trace",
		Handler:     "text",
		GoRoutineID: &fls,
	})

	lg.Trace("trace message", "field1", "value1", "field2", "value2")
	lg.Debug("debug message", "field1", "value1", "field2", "value2")
	lg.Info("info message", "field1", "value1", "field2", "value2")
	lg.Warn("warn message", "field1", "value1", "field2", "value2")
	lg.Error("error message", "field1", "value1", "field2", "value2")

	// Output:
	// 1970-01-01T00:00:00.000Z TRACE trace message             logger=/ field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z DEBUG debug message             logger=/ field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z INFO  info message              logger=/ field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z WARN  warn message              logger=/ field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z ERROR error message             logger=/ field1=value1 field2=value2
}
