package log

import (
	"fmt"
	"io"
	"strings"
	"time"

	apex "github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/utc-go"
	"github.com/rs/zerolog"
)

// TraceEvent returns a zerolog event for zero-allocation logging at Trace level. When the Trace
// level is disabled the returned event is a no-op and all chained method calls are elided. Commit
// the entry by calling Msg() or Send().
func (l *Log) TraceEvent() *zerolog.Event {
	return l.get().zl.Trace()
}

// DebugEvent returns a zerolog event for zero-allocation logging at Debug level. When the Debug
// level is disabled the returned event is a no-op.
func (l *Log) DebugEvent() *zerolog.Event {
	return l.get().zl.Debug()
}

// InfoEvent returns a zerolog event for zero-allocation logging at Info level.
func (l *Log) InfoEvent() *zerolog.Event {
	return l.get().zl.Info()
}

// WarnEvent returns a zerolog event for zero-allocation logging at Warn level.
func (l *Log) WarnEvent() *zerolog.Event {
	return l.get().zl.Warn()
}

// ErrorEvent returns a zerolog event for zero-allocation logging at Error level.
func (l *Log) ErrorEvent() *zerolog.Event {
	return l.get().zl.Error()
}

// FatalEvent returns a zerolog event for zero-allocation logging at Fatal level.
// As with the regular Fatal method, the process exits after the entry is written.
func (l *Log) FatalEvent() *zerolog.Event {
	return l.get().zl.Fatal()
}

// apexToZerologLevel converts an apex log level to the corresponding zerolog level.
func apexToZerologLevel(l apex.Level) zerolog.Level {
	switch l {
	case apex.TraceLevel:
		return zerolog.TraceLevel
	case apex.DebugLevel:
		return zerolog.DebugLevel
	case apex.InfoLevel:
		return zerolog.InfoLevel
	case apex.WarnLevel:
		return zerolog.WarnLevel
	case apex.ErrorLevel:
		return zerolog.ErrorLevel
	case apex.FatalLevel:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// utcMillisTimestamp is a zerolog hook that appends a UTC timestamp with millisecond precision to
// every event. It uses utc.Now() as the time source, which is mockable in tests via utc.MockNowFn.
type utcMillisTimestamp struct{}

func (h utcMillisTimestamp) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	// Use e.Time rather than e.Str(..., utc.Now().String()): the latter builds an intermediate Go
	// string (a 24-byte heap allocation per event), while e.Time appends the formatted time directly
	// into the event's buffer with zero allocations, using zerolog.TimeFieldFormat (see init below).
	e.Time(zerolog.TimestampFieldName, utc.Now().Time)
}

func init() {
	// Emit timestamps in ISO 8601 UTC with millisecond precision, matching utc.UTC.String().
	zerolog.TimeFieldFormat = utc.ISO8601
}

// newZerologLogger creates a zerolog.Logger that writes to w. The output format is configured to
// approximate the given apex handler type so that fast-path and regular-path output are visually
// consistent. For text and raw handlers a ConsoleWriter is used; for console a coloured
// ConsoleWriter is used; for all other handlers (json, discard, memory) output is plain JSON.
//
// Timestamps are emitted in UTC with millisecond precision via utcMillisTimestamp. The logger
// name field is added to the context for all handler types except console and memory, matching
// the behaviour of apex's defaultFields.
func newZerologLogger(w io.Writer, level apex.Level, handlerType, loggerName string) zerolog.Logger {
	var zw io.Writer
	switch handlerType {
	case "console":
		zw = zerolog.ConsoleWriter{
			Out:           w,
			TimeFormat:    "2006-01-02T15:04:05.000Z07:00",
			TimeLocation:  time.UTC,
			FormatLevel:   zlFormatLevel,
			FormatMessage: zlFormatMessage,
		}
	case "text", "raw":
		zw = zerolog.ConsoleWriter{
			Out:           w,
			TimeFormat:    "2006-01-02T15:04:05.000Z07:00",
			TimeLocation:  time.UTC,
			NoColor:       true,
			FormatLevel:   zlFormatLevel,
			FormatMessage: zlFormatMessage,
		}
	default: // "json", "discard", "memory" — plain JSON (io.Discard is passed for discard/memory)
		zw = w
	}

	zl := zerolog.New(zw).Level(apexToZerologLevel(level)).Hook(utcMillisTimestamp{})

	// Mirror apex's defaultFields: logger name is omitted for console and memory handlers.
	switch handlerType {
	case "console", "memory":
	default:
		if loggerName != "" {
			zl = zl.With().Str("logger", loggerName).Logger()
		}
	}

	return zl
}

// zlFormatLevel formats a zerolog level token to match the apex text handler's output (e.g. "INFO ").
func zlFormatLevel(i interface{}) string {
	l, _ := i.(string)
	switch l {
	case "trace":
		return "TRACE"
	case "debug":
		return "DEBUG"
	case "info":
		return "INFO "
	case "warn":
		return "WARN "
	case "error":
		return "ERROR"
	case "fatal":
		return "FATAL"
	default:
		return strings.ToUpper(fmt.Sprintf("%-5s", l))
	}
}

// zlFormatMessage pads the message to 25 characters, matching the apex text handler's %-25s format.
func zlFormatMessage(i interface{}) string {
	return fmt.Sprintf("%-25s", i)
}
