package log

import "time"

// SetDefault sets the default configuration and creates the default log based on that configuration.
func SetDefault(c *Config) {
	getLogRoot().setDefault(c)
}

// Get returns the named logger for the given path. Loggers are organized in a
// hierarchy (tree) defined by their paths. Paths use a forward slash '/' as
// separator (e.g. /eluvio/util/json). Loggers inherit attributes
// of their parent loggers.
// A logger's path is added to every log entry as a field: logger=/eluvio/util/json
func Get(path string) *Log {
	return getLogRoot().Get(path)
}

// Root retrieves the root logger - same as Get("/")
func Root() *Log {
	return Get("/")
}

func CloseLogFiles() {
	getLogRoot().closeLogs()
}

// Trace logs the given message at the Trace level.
func Trace(msg string, fields ...interface{}) {
	def().Trace(msg, fields...)
}

// Debug logs the given message at the Debug level.
func Debug(msg string, fields ...interface{}) {
	def().Debug(msg, fields...)
}

// Info logs the given message at the Info level.
func Info(msg string, fields ...interface{}) {
	def().Info(msg, fields...)
}

// Warn logs the given message at the Warn level.
func Warn(msg string, fields ...interface{}) {
	def().Warn(msg, fields...)
}

// Error logs the given message at the Error level.
func Error(msg string, fields ...interface{}) {
	def().Error(msg, fields...)
}

// Fatal logs the given message at the Fatal level.
func Fatal(msg string, fields ...interface{}) {
	def().Fatal(msg, fields...)
}

// IsTrace returns true if the logger logs in Trace level.
func IsTrace() bool {
	return def().IsTrace()
}

// IsDebug returns true if the logger logs in Debug level.
func IsDebug() bool {
	return def().IsDebug()
}

// IsInfo returns true if the logger logs in Info level.
func IsInfo() bool {
	return def().IsInfo()
}

// IsWarn returns true if the logger logs in Warn level.
func IsWarn() bool {
	return def().IsWarn()
}

// IsError returns true if the logger logs in Error level.
func IsError() bool {
	return def().IsError()
}

// IsFatal returns true if the logger logs in Fatal level.
func IsFatal() bool {
	return def().IsFatal()
}

// Throttle returns a decorator of this log that limits the number of log messages emitted per period. The decorator is
// tied to the given throttle key - different keys result in separate instances. The decorator logs the first message
// and suppresses all subsequent messages that are logged within the provided period (5 seconds by default). The first
// message in the new period is logged again, with an indication of the number of entries that were suppressed.
//
//	1970-01-01T00:00:00.000Z WARN  failed to connect         attempt=1 error=connect error
//	1970-01-01T00:00:01.000Z WARN  failed to connect         attempt=11 suppressed=9 throttle_period=1s error=connect error
//	1970-01-01T00:00:02.000Z WARN  failed to connect         attempt=21 suppressed=9 throttle_period=1s error=connect error
func Throttle(key string, period ...time.Duration) Throttled {
	return def().Throttle(key, period...)
}
