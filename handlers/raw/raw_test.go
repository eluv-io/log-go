package raw_test

import (
	"fmt"

	"github.com/eluv-io/log-go"
	"github.com/eluv-io/utc-go"
)

func ExampleHandler() {
	defer utc.MockNow(utc.UnixMilli(0))()

	fls := false
	lg := log.New(&log.Config{
		Level:       "trace",
		Handler:     "raw",
		GoRoutineID: &fls,
	})

	lg.Trace("trace message", "field1", "value1", "field2", "value2")
	lg.Debug("debug message", "field1", "value1", "field2", "value2")
	lg.Info("info message", "field1", "value1", "field2", "value2")
	lg.Warn("warn message", "field1", "value1", "field2", "value2")
	lg.Error("error message", "field1", "value1", "field2", "value2")

	fmt.Println()

	lg.Trace("trace message", "field1", "value1", "field2", "value2", "raw", "raw string")
	lg.Debug("debug message", "field1", "value1", "field2", "value2", "raw", "raw string")
	lg.Info("info message", "field1", "value1", "field2", "value2", "raw", "raw string")
	lg.Warn("warn message", "field1", "value1", "field2", "value2", "raw", "raw string")
	lg.Error("error message", "field1", "value1", "field2", "value2", "raw", "raw string")

	// Output:
	// 1970-01-01T00:00:00.000Z trace message             field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z debug message             field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z info message              field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z warn message              field1=value1 field2=value2
	// 1970-01-01T00:00:00.000Z error message             field1=value1 field2=value2
	//
	// 1970-01-01T00:00:00.000Z trace message             field1=value1 field2=value2
	// raw string
	//
	// 1970-01-01T00:00:00.000Z debug message             field1=value1 field2=value2
	// raw string
	//
	// 1970-01-01T00:00:00.000Z info message              field1=value1 field2=value2
	// raw string
	//
	// 1970-01-01T00:00:00.000Z warn message              field1=value1 field2=value2
	// raw string
	//
	// 1970-01-01T00:00:00.000Z error message             field1=value1 field2=value2
	// raw string
	//
}
