// Package text implements a development-friendly textual handler.
package text

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/apex/log"
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

	_, _ = fmt.Fprintf(sb, "%s %-5s %-25s", utc.Now().String(), e.Level.String(), e.Message)

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
