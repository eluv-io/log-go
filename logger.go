package log

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/modern-go/gls"
	"gopkg.in/natefinch/lumberjack.v2"

	apex "github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/errors-go"
)

// logger is the actual implementation of a Log
type logger struct {
	log        apex.Interface     // log is the logger decorated with the logger name field
	name       string             // name is the logger's name when created through Get()
	config     *Config            // the current config
	lumberjack *lumberjack.Logger // io.WriteCloser that writes to the specified filename.
}

func copyApexLogger(log apex.Interface) apex.Interface {
	switch al := log.(type) {
	case *apex.Logger:
		return &apex.Logger{
			Handler: al.Handler,
			Level:   al.Level,
		}
	case *apex.Entry:
		apx := &apex.Logger{
			Handler: al.Logger.Handler,
			Level:   al.Logger.Level,
		}
		return apex.NewEntry(apx).WithFields(al.MergedFields())
	default:
		panic(errors.Str(fmt.Sprintf("copyApexLogger: unknown type %v", reflect.TypeOf(log))))
	}
}

func (l *logger) copy(modFns ...func(l *logger)) *logger {
	ret := &logger{
		log:        copyApexLogger(l.log),
		name:       l.name,
		config:     l.config,
		lumberjack: l.lumberjack,
	}
	for _, fn := range modFns {
		fn(ret)
	}
	return ret
}

func (l *logger) logger() *apex.Logger {
	switch al := l.log.(type) {
	case *apex.Logger:
		return al
	case *apex.Entry:
		return al.Logger
	default:
		panic(errors.Str(fmt.Sprintf("logger: unknown type %v", reflect.TypeOf(l.log))))
	}
}

func (l *logger) handler() apex.Handler {
	return l.logger().Handler
}

// IsTrace returns true if the logger logs in Trace level.
func (l *logger) IsTrace() bool {
	return l.logger().Level <= apex.TraceLevel
}

// IsDebug returns true if the logger logs in Debug level.
func (l *logger) IsDebug() bool {
	return l.logger().Level <= apex.DebugLevel
}

// IsInfo returns true if the logger logs in Info level.
func (l *logger) IsInfo() bool {
	return l.logger().Level <= apex.InfoLevel
}

// IsWarn returns true if the logger logs in Warn level.
func (l *logger) IsWarn() bool {
	return l.logger().Level <= apex.WarnLevel
}

// IsError returns true if the logger logs in Error level.
func (l *logger) IsError() bool {
	return l.logger().Level <= apex.ErrorLevel
}

// IsFatal returns true if the logger logs in Fatal level.
func (l *logger) IsFatal() bool {
	return l.logger().Level <= apex.FatalLevel
}

// Trace logs the given message at the Trace level.
func (l *logger) Trace(msg string, fields ...interface{}) {
	metrics().Debug(l.name)
	if l.IsTrace() {
		l.log.Trace(msg, l.fields(fields)...)
	}
}

// Debug logs the given message at the Debug level.
func (l *logger) Debug(msg string, fields ...interface{}) {
	metrics().Debug(l.name)
	if l.IsDebug() {
		l.log.Debug(msg, l.fields(fields)...)
	}
}

// Info logs the given message at the Info level.
func (l *logger) Info(msg string, fields ...interface{}) {
	metrics().Info(l.name)
	if l.IsInfo() {
		l.log.Info(msg, l.fields(fields)...)
	}
}

// Warn logs the given message at the Warn level.
func (l *logger) Warn(msg string, fields ...interface{}) {
	metrics().Warn(l.name)
	if l.IsWarn() {
		l.log.Warn(msg, l.fields(fields)...)
	}
}

// Error logs the given message at the Error level.
func (l *logger) Error(msg string, fields ...interface{}) {
	metrics().Error(l.name)
	if l.IsError() {
		l.log.Error(msg, l.fields(fields)...)
	}
}

// Fatal logs the given message at the Fatal level.
func (l *logger) Fatal(msg string, fields ...interface{}) {
	l.log.Fatal(msg, l.fields(fields)...)
}

func (l *logger) fields(args []interface{}) []interface{} {
	if l.config.GoRoutineID != nil && *l.config.GoRoutineID {
		args = append(args, "gid", goID())
	}

	if l.config.Caller != nil && *l.config.Caller {
		args = append(args, "caller", caller(2))
	}

	return args
}

// goID returns the goroutine id of current goroutine
func goID() int64 {
	return gls.GoID()
}

// caller returns the file and line number of the caller, formatted as "file:line".
func caller(framesToSkip int) string {
	_, file, line, ok := runtime.Caller(framesToSkip + 2) // +2 to account for call to *logger
	if !ok {
		return "?"
	}

	files := strings.Split(file, "/")
	file = files[len(files)-1]

	return fmt.Sprintf("%s:%d", file, line)
}
