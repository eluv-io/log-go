package log_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/log-go"
)

func TestLevels(t *testing.T) {
	assertLevelTrace(t, tl("trace"))
	assertLevelDebug(t, tl("debug"))
	assertLevelInfo(t, tl("info"))
	assertLevelWarn(t, tl("warn"))
	assertLevelError(t, tl("error"))
	assertLevelFatal(t, tl("fatal"))
}

func tl(level string) *log.Log {
	return log.New(
		&log.Config{
			Handler: "memory",
			Level:   level,
		})
}

func assertLevelTrace(t *testing.T, logger *log.Log) {
	assert.True(t, logger.IsTrace())
}

func assertLevelDebug(t *testing.T, logger *log.Log) {
	assert.True(t, logger.IsDebug())
	assert.False(t, logger.IsTrace())
}

func assertLevelInfo(t *testing.T, logger *log.Log) {
	assert.True(t, logger.IsInfo())
	assert.False(t, logger.IsDebug())
}

func assertLevelWarn(t *testing.T, logger *log.Log) {
	assert.True(t, logger.IsWarn())
	assert.False(t, logger.IsInfo())
}

func assertLevelError(t *testing.T, logger *log.Log) {
	assert.True(t, logger.IsError())
	assert.False(t, logger.IsWarn())
}

func assertLevelFatal(t *testing.T, logger *log.Log) {
	assert.True(t, logger.IsFatal())
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

func TestLevelPropagation(t *testing.T) {
	c := log.Config{
		Level:   "info",
		Handler: "memory",
		Named: map[string]*log.Config{
			"/api": {
				Level: "debug",
			},
			"/db": {
				Level: "warn",
			},
		},
	}
	log.SetDefault(&c)
	// handler, ok := log.Get("").Handler().(*memory.Handler)
	// require.True(t, ok)

	root := log.Root()

	api := log.Get("/api")
	ep1 := log.Get("/api/ep1") // inherits from /api
	ep2 := log.Get("/api/ep2") // inherits from /api
	db := log.Get("/db")

	assertLevelInfo(t, root)
	assertLevelDebug(t, api)
	assertLevelDebug(t, ep1)
	assertLevelDebug(t, ep2)
	assertLevelWarn(t, db)

	ep2.SetWarn()

	assertLevelInfo(t, root)
	assertLevelDebug(t, api)
	assertLevelDebug(t, ep1)
	assertLevelWarn(t, ep2)
	assertLevelWarn(t, db)

	root.SetError()

	assertLevelError(t, root)
	assertLevelError(t, api)
	assertLevelError(t, ep1)
	assertLevelError(t, ep2)
	assertLevelError(t, db)

	api.SetTrace()

	assertLevelError(t, root)
	assertLevelTrace(t, api)
	assertLevelTrace(t, ep1)
	assertLevelTrace(t, ep2)
	assertLevelError(t, db)

	snl := log.Get("/some/new/logger")
	assertLevelError(t, snl) // inherited from root

	ep3 := log.Get("/api/ep3")
	assertLevelTrace(t, ep3) // inherited from /api (which was set to trace above)
}
