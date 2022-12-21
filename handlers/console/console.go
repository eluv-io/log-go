// Package console implements a development-friendly textual handler.
package console

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/utc-go"
)

// Default handler outputting to stderr.
var Default = New(os.Stderr)

// colors.
const (
	red     = 31
	yellow  = 33
	blue    = 34
	magenta = 35
	gray    = 37
)

const (
	normal = 0
	bold   = 1
)

// Colors mapping.
var Colors = [...]int{
	log.TraceLevel: gray,
	log.DebugLevel: yellow,
	log.InfoLevel:  blue,
	log.WarnLevel:  magenta,
	log.ErrorLevel: red,
	log.FatalLevel: red,
}

// Intensities color mapping.
var Intensities = [...]int{
	log.TraceLevel: normal,
	log.DebugLevel: normal,
	log.InfoLevel:  normal,
	log.WarnLevel:  normal,
	log.ErrorLevel: normal,
	log.FatalLevel: bold,
}

var Levels = [...]string{
	log.TraceLevel: "TRCE",
	log.DebugLevel: "DBG ",
	log.InfoLevel:  "    ",
	log.WarnLevel:  "WARN",
	log.ErrorLevel: "ERR!",
	log.FatalLevel: "FATL",
}

// Handler implementation.
type Handler struct {
	start         utc.UTC
	noColor       bool
	mu            sync.Mutex
	Writer        io.Writer
	UseTimestamps bool
}

// New creates a new console handler.
func New(w io.Writer) *Handler {
	return &Handler{
		start:  utc.Now(),
		Writer: w,
	}
}

func (h *Handler) WithColor(colored bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.noColor = !colored
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {

	sb := &strings.Builder{}

	color := Colors[e.Level]
	intensity := Intensities[e.Level]
	colored := !h.noColor
	level := Levels[e.Level]

	d := utc.Since(h.start)
	ts := d / time.Second
	tms := (d - ts*time.Second) / time.Millisecond
	if colored {
		if h.UseTimestamps {
			_, _ = fmt.Fprintf(sb, "%s \033[%d;%dm%-5s\033[0m %-20s", utc.Now().String(), intensity, color, level, e.Message)
		} else {
			_, _ = fmt.Fprintf(sb, "% 4d.%03d \033[%d;%dm%-5s\033[0m %-20s", ts, tms, intensity, color, level, e.Message)
		}
		//fmt.Fprintf(sb, "\033[%dm%-1s\033[0m % 4d.%03d %-20s", color, level, ts, tms, e.Message)
	} else {
		_, _ = fmt.Fprintf(sb, "% 4d.%03d %-5s %-20s", ts, tms, level, e.Message)
	}

	for _, field := range e.Fields {
		if colored {
			_, _ = fmt.Fprintf(sb, " %s=\033[%d;%dm%v\033[0m", field.Name, intensity, color, field.Value)
		} else {
			_, _ = fmt.Fprintf(sb, " %s=%v", field.Name, field.Value)
		}
	}

	_, _ = fmt.Fprintln(sb)

	h.mu.Lock()
	defer h.mu.Unlock()

	_, _ = h.Writer.Write([]byte(sb.String()))

	return nil
}
