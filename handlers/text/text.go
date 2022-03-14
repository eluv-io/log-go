// Package text implements a development-friendly textual handler.
package text

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/utc-go"
)

// Default handler outputting to stderr.
var Default = New(os.Stderr)

// Levels is the array of strings used for printing the log level
var Levels = [...]string{
	log.TraceLevel: "TRACE",
	log.DebugLevel: "DEBUG",
	log.InfoLevel:  "INFO ",
	log.WarnLevel:  "WARN ",
	log.ErrorLevel: "ERROR",
	log.FatalLevel: "FATAL",
}

// Handler implementation.
type Handler struct {
	mu     sync.Mutex
	Writer io.Writer
}

// New creates a new text handler
func New(w io.Writer) *Handler {
	return &Handler{
		Writer: w,
	}
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {
	level := Levels[e.Level]

	sb := &strings.Builder{}

	_, _ = fmt.Fprintf(sb, "%s %s %-25s", utc.Now().String(), level, e.Message)

	// print error field at the end, since they often have nested errors that
	// are printed on separate lines
	var err interface{}
	for _, field := range e.Fields {
		if field.Name == "error" {
			err = field.Value
		} else {
			_, _ = fmt.Fprintf(sb, " %s=%v", field.Name, field.Value)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(sb, " %s=%v", "error", err)
	}

	_, _ = fmt.Fprintln(sb)

	h.mu.Lock()
	defer h.mu.Unlock()

	_, _ = h.Writer.Write([]byte(sb.String()))

	return nil
}
