// Package log provides a minimal wrapper around an arbitrary third-party
// logging library.
//
// The wrapper abstracts from a concrete logging implementation in order to
// allow painless and seamless switch-over to alternative implementations in
// the future. It provides the following functionality:
//
// - Levelled logging through the Debug, Info, Warn, Error and Fatal log methods.
//
// - Structured logging using named fields (i.e. key-value pairs) for
// simplified log parsing. Depending on the underlying log implementation,
// log entries can be written in json format, simplifying log parsing even
// further.
//
// - Super-simple function signatures. All log functions take a message string
// and an arbitrary number of additional arguments interpreted as fields
// (key-value pairs) for the structured logging.
//
// The log message should be static (i.e. it should not contain any dynamic
// content), so that log parsing remains simple. Any dynamic data should be
// added as additional fields.
//
// Example:
//
//	// conventional plain-text log
//	log.Infof("uploading %s for %s", filename, user)
//	log.Warnf("upload failed %s for %s: %s", filename, user, err)
//
//	// structured logging
//	log.Info("uploading", "file", filename, "user", user)
//	log.Warn("upload failed", "file", filename, "user", user, "error", err)
//
//	// same as above: an error can be logged without specifying a key ("error" will be used as key)
//	log.Warn("upload failed", "file", filename, "user", user, err)
//
// Some background on structured logging: https://medium.com/@tjholowaychuk/apex-log-e8d9627f4a9a
package log

import (
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"

	"gopkg.in/natefinch/lumberjack.v2"

	apex "github.com/eluv-io/apexlog-go"
)

// New creates a new root Logger
func New(c *Config) *Log {
	return newLog(c, defaultFields(c, "/"), nil)
}

func NewLumberjackLogger(c *LumberjackConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   c.Filename,
		MaxSize:    c.MaxSize,
		MaxAge:     c.MaxAge,
		MaxBackups: c.MaxBackups,
		LocalTime:  c.LocalTime,
		Compress:   c.Compress,
	}
}

// Log provides the fundamental logging functions. It's implemented as a wrapper around the actual logger implementation
// that allows concurrency-safe modification (replacement) of the underlying logger.
type Log struct {
	lw atomic.Pointer[logger]
}

func (l *Log) get() *logger {
	return l.lw.Load()
}

func (l *Log) set(lg *logger) {
	l.lw.Store(lg)
}

// Handler returns the handler of this Log
// Modifying the returned handler is not safe for concurrent calls.
func (l *Log) Handler() apex.Handler {
	return l.get().handler()
}

func (l *Log) updateFrom(nl *Log) {
	l.lw.Store(nl.lw.Load())
}

// Trace logs the given message at the Trace level.
func (l *Log) Trace(msg string, fields ...interface{}) {
	l.get().Trace(msg, fields...)
}

// Debug logs the given message at the Debug level.
func (l *Log) Debug(msg string, fields ...interface{}) {
	l.get().Debug(msg, fields...)
}

// Info logs the given message at the Info level.
func (l *Log) Info(msg string, fields ...interface{}) {
	l.get().Info(msg, fields...)
}

// Warn logs the given message at the Warn level.
func (l *Log) Warn(msg string, fields ...interface{}) {
	l.get().Warn(msg, fields...)
}

// Error logs the given message at the Error level.
func (l *Log) Error(msg string, fields ...interface{}) {
	l.get().Error(msg, fields...)
}

// Fatal logs the given message at the Fatal level.
func (l *Log) Fatal(msg string, fields ...interface{}) {
	l.get().Fatal(msg, fields...)
}

// IsTrace returns true if the logger logs in Trace level.
func (l *Log) IsTrace() bool {
	return l.get().IsTrace()
}

// IsDebug returns true if the logger logs in Debug level.
func (l *Log) IsDebug() bool {
	return l.get().IsDebug()
}

// IsInfo returns true if the logger logs in Info level.
func (l *Log) IsInfo() bool {
	return l.get().IsInfo()
}

// IsWarn returns true if the logger logs in Warn level.
func (l *Log) IsWarn() bool {
	return l.get().IsWarn()
}

// IsError returns true if the logger logs in Error level.
func (l *Log) IsError() bool {
	return l.get().IsError()
}

// IsFatal returns true if the logger logs in Fatal level.
func (l *Log) IsFatal() bool {
	return l.get().IsFatal()
}

// Name returns the name of this logger
func (l *Log) Name() string {
	return l.get().name
}

func (l *Log) Level() string {
	return l.get().logger().Level.String()
}

// SetLevel sets the log level according to the given string.
func (l *Log) SetLevel(level string) {
	lvl, err := apex.ParseLevel(level)
	if err != nil {
		return
	}
	l.setLogLevel(lvl)
}

// SetTrace sets the log level to Trace.
func (l *Log) SetTrace() {
	l.setLogLevel(apex.TraceLevel)
}

// SetDebug sets the log level to Debug.
func (l *Log) SetDebug() {
	l.setLogLevel(apex.DebugLevel)
}

// SetInfo sets the log level to Info.
func (l *Log) SetInfo() {
	l.setLogLevel(apex.InfoLevel)
}

// SetWarn sets the log level to Warn.
func (l *Log) SetWarn() {
	l.setLogLevel(apex.WarnLevel)
}

// SetError sets the log level to Error.
func (l *Log) SetError() {
	l.setLogLevel(apex.ErrorLevel)
}

// SetFatal sets the log level to Fatal.
func (l *Log) SetFatal() {
	l.setLogLevel(apex.FatalLevel)
}

func (l *Log) getLogRoot() *logRoot {
	return getLogRoot()
}

func (l *Log) setLogLevel(level apex.Level) {
	setLevel := func(logCopy *logger) {
		logCopy.logger().Level = level
		logCopy.config.Level = level.String()
	}
	logName := l.get().name

	root := l.getLogRoot()
	root.doLocked(func(r *logRoot) {
		for name, log := range r.named {
			oldLogger := log.get()
			if strings.HasPrefix(name, logName) {
				newLogger := oldLogger.copy(setLevel)
				log.set(newLogger)
			}
		}
		l.set(l.get().copy(setLevel))
	})
}

// Call invokes the function and simply logs if an error occurs. Useful when
// deferring a call like io.Close:
//
//	defer log.Call(reader.Close, "package.Func")
//
// Optionally provide a log function:
//
//	log.Call(reader.Close, "package.Func", mylog.Debug)
func (l *Log) Call(f func() error, msg string, log ...func(msg string, fields ...interface{})) {
	if f != nil {
		if err := f(); err != nil {
			fname := "unknown"
			if ffp := runtime.FuncForPC(reflect.ValueOf(f).Pointer()); ffp != nil {
				fname = ffp.Name()
			}
			if len(log) > 0 {
				log[0](msg, err, "func", fname)
			} else {
				l.Info(msg, err, "func", fname)
			}
		}
	}
}
