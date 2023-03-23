package console_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eluv-io/log-go"
	"github.com/eluv-io/log-go/handlers/console"
	"github.com/eluv-io/utc-go"
)

func TestHandler(t *testing.T) {
	defer utc.MockNow(utc.UnixMilli(0))()

	tests := []struct {
		name   string
		caller bool
		adapt  func(h *console.Handler)
		want   string
	}{
		{
			name:   "default: offset, color",
			caller: false,
			adapt:  func(h *console.Handler) {},
			want: "" +
				"   0.000 \033[0;37mTRCE \033[0m trace message        field1=\033[0;37mvalue1\033[0m field2=\033[0;37mvalue2\033[0m\n" +
				"   0.000 \033[0;33mDBG  \033[0m debug message        field1=\033[0;33mvalue1\033[0m field2=\033[0;33mvalue2\033[0m\n" +
				"   0.000 \033[0;34m     \033[0m info message         field1=\033[0;34mvalue1\033[0m field2=\033[0;34mvalue2\033[0m\n" +
				"   0.000 \033[0;35mWARN \033[0m warn message         field1=\033[0;35mvalue1\033[0m field2=\033[0;35mvalue2\033[0m\n" +
				"   0.000 \033[0;31mERR! \033[0m error message        field1=\033[0;31mvalue1\033[0m field2=\033[0;31mvalue2\033[0m\n",
		},
		{
			name:   "offset, no color",
			caller: false,
			adapt: func(h *console.Handler) {
				h.WithColor(false)
			},
			want: "" +
				"   0.000 TRCE  trace message        field1=value1 field2=value2\n" +
				"   0.000 DBG   debug message        field1=value1 field2=value2\n" +
				"   0.000       info message         field1=value1 field2=value2\n" +
				"   0.000 WARN  warn message         field1=value1 field2=value2\n" +
				"   0.000 ERR!  error message        field1=value1 field2=value2\n",
		},
		{
			name:   "timestamp, color",
			caller: false,
			adapt: func(h *console.Handler) {
				h.WithTimestamps(true)
			},
			want: "" +
				"1970-01-01T00:00:00.000Z \033[0;37mTRCE \033[0m trace message        field1=\033[0;37mvalue1\033[0m field2=\033[0;37mvalue2\033[0m\n" +
				"1970-01-01T00:00:00.000Z \033[0;33mDBG  \033[0m debug message        field1=\033[0;33mvalue1\033[0m field2=\033[0;33mvalue2\033[0m\n" +
				"1970-01-01T00:00:00.000Z \033[0;34m     \033[0m info message         field1=\033[0;34mvalue1\033[0m field2=\033[0;34mvalue2\033[0m\n" +
				"1970-01-01T00:00:00.000Z \033[0;35mWARN \033[0m warn message         field1=\033[0;35mvalue1\033[0m field2=\033[0;35mvalue2\033[0m\n" +
				"1970-01-01T00:00:00.000Z \033[0;31mERR! \033[0m error message        field1=\033[0;31mvalue1\033[0m field2=\033[0;31mvalue2\033[0m\n",
		},
		{
			name:   "timestamp, no color",
			caller: false,
			adapt: func(h *console.Handler) {
				h.WithTimestamps(true).WithColor(false)
			},
			want: "" +
				"1970-01-01T00:00:00.000Z TRCE  trace message        field1=value1 field2=value2\n" +
				"1970-01-01T00:00:00.000Z DBG   debug message        field1=value1 field2=value2\n" +
				"1970-01-01T00:00:00.000Z       info message         field1=value1 field2=value2\n" +
				"1970-01-01T00:00:00.000Z WARN  warn message         field1=value1 field2=value2\n" +
				"1970-01-01T00:00:00.000Z ERR!  error message        field1=value1 field2=value2\n",
		},
		{
			name:   "timestamp, color, caller",
			caller: true,
			adapt: func(h *console.Handler) {
				h.WithTimestamps(true).WithColor(false)
			},
			want: "" +
				"1970-01-01T00:00:00.000Z TRCE  trace message        field1=value1 field2=value2 caller=console_test.go:103\n" +
				"1970-01-01T00:00:00.000Z DBG   debug message        field1=value1 field2=value2 caller=console_test.go:104\n" +
				"1970-01-01T00:00:00.000Z       info message         field1=value1 field2=value2 caller=console_test.go:105\n" +
				"1970-01-01T00:00:00.000Z WARN  warn message         field1=value1 field2=value2 caller=console_test.go:106\n" +
				"1970-01-01T00:00:00.000Z ERR!  error message        field1=value1 field2=value2 caller=console_test.go:107\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fls := false
			lg := log.New(&log.Config{
				Level:       "trace",
				Handler:     "console",
				GoRoutineID: &fls,
				Caller:      test.caller,
			})

			buf := &bytes.Buffer{}
			handler := lg.Handler().(*console.Handler)
			handler.Writer = buf
			test.adapt(handler)

			lg.Trace("trace message", "field1", "value1", "field2", "value2")
			lg.Debug("debug message", "field1", "value1", "field2", "value2")
			lg.Info("info message", "field1", "value1", "field2", "value2")
			lg.Warn("warn message", "field1", "value1", "field2", "value2")
			lg.Error("error message", "field1", "value1", "field2", "value2")

			require.Equal(t, test.want, buf.String())
		})
	}

}
