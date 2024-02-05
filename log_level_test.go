package log_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/log-go"
)

func TestLevels(t *testing.T) {
	assertLevel(t, tl("debug"), true, true, true, true, true)
	assertLevel(t, tl("info"), false, true, true, true, true)
	assertLevel(t, tl("warn"), false, false, true, true, true)
	assertLevel(t, tl("error"), false, false, false, true, true)
	assertLevel(t, tl("fatal"), false, false, false, false, true)
}

func tl(level string) *log.Log {
	return log.New(
		&log.Config{
			Handler: "memory",
			Level:   level,
		})
}

func assertLevel(t *testing.T, logger *log.Log, isDebug, isInfo, isWarn, isError, isFatal bool) {
	assert.Equal(t, isDebug, logger.IsDebug())
	assert.Equal(t, isInfo, logger.IsInfo())
	assert.Equal(t, isWarn, logger.IsWarn())
	assert.Equal(t, isError, logger.IsError())
	assert.Equal(t, isFatal, logger.IsFatal())
}

func TestLevel(t *testing.T) {

	newLogConfig := func(level string) *log.Config {
		c := log.NewConfig()
		c.Level = level
		return c
	}
	type testCase struct {
		level string
		fn    []func() bool
		do    func()
	}
	for _, tc := range []*testCase{
		{
			level: "trace",
			fn: []func() bool{
				func() bool { return log.Root().IsTrace() },
				func() bool { return log.IsTrace() },
			},
			do: func() { log.Trace("trace") },
		},
		{
			level: "debug",
			fn: []func() bool{
				func() bool { return log.Root().IsDebug() },
				func() bool { return log.IsDebug() },
			},
			do: func() { log.Debug("debug") },
		},
		{
			level: "info",
			fn: []func() bool{
				func() bool { return log.Root().IsInfo() },
				func() bool { return log.IsInfo() },
			},
			do: func() { log.Info("info") },
		},
		{
			level: "warn",
			fn: []func() bool{
				func() bool { return log.Root().IsWarn() },
				func() bool { return log.IsWarn() },
			},
			do: func() { log.Warn("warn") },
		},
		{
			level: "error",
			fn: []func() bool{
				func() bool { return log.Root().IsError() },
				func() bool { return log.IsError() },
			},
			do: func() { log.Error("error") },
		},
		{
			level: "fatal",
			fn: []func() bool{
				func() bool { return log.Root().IsFatal() },
				func() bool { return log.IsFatal() },
			},
		},
	} {
		c := newLogConfig(tc.level)
		log.SetDefault(c)
		for _, fn := range tc.fn {
			require.True(t, fn(), "failed at %v", tc.level)
		}
		if tc.do != nil {
			tc.do()
		}
	}

}
