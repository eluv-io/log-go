// Package raw is like handlers/text, but omits the "log level" field and prints
// the "raw" field without label on a separate line.
package raw

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

// Handler implementation.
type Handler struct {
	mu     sync.Mutex
	Writer io.Writer
}

// New handler.
func New(w io.Writer) *Handler {
	return &Handler{
		Writer: w,
	}
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {
	sb := &strings.Builder{}

	_, _ = fmt.Fprintf(sb, "%s %-25s", utc.Now().String(), e.Message)

	for _, field := range e.Fields {
		switch field.Name {
		case "raw":
		case "logger":
		default:
			_, _ = fmt.Fprintf(sb, " %s=%v", field.Name, field.Value)
		}
	}

	sb.Write([]byte{'\n'})
	raw := e.Fields.Get("raw")
	if raw != "" {
		_, _ = fmt.Fprintf(sb, "\n%v\n", raw)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	_, _ = h.Writer.Write([]byte(sb.String()))

	return nil
}
