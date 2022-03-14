package console_test

import (
	"bytes"
	"testing"

	"github.com/eluv-io/log-go"
	"github.com/eluv-io/log-go/handlers/console"
	"github.com/eluv-io/utc-go"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	//func ExampleHandler() {
	defer utc.MockNow(utc.UnixMilli(0))()

	fls := false
	lg := log.New(&log.Config{
		Level:       "trace",
		Handler:     "console",
		GoRoutineID: &fls,
	})

	buf := &bytes.Buffer{}
	lg.Handler().(*console.Handler).Writer = buf

	lg.Trace("trace message", "field1", "value1", "field2", "value2")
	lg.Debug("debug message", "field1", "value1", "field2", "value2")
	lg.Info("info message", "field1", "value1", "field2", "value2")
	lg.Warn("warn message", "field1", "value1", "field2", "value2")
	lg.Error("error message", "field1", "value1", "field2", "value2")

	exp := "" +
		"   0.001 \033[0;37mTRCE \033[0m trace message        field1=\033[0;37mvalue1\033[0m field2=\033[0;37mvalue2\033[0m\n" +
		"   0.001 \033[0;33mDBG  \033[0m debug message        field1=\033[0;33mvalue1\033[0m field2=\033[0;33mvalue2\033[0m\n" +
		"   0.001 \033[0;34m     \033[0m info message         field1=\033[0;34mvalue1\033[0m field2=\033[0;34mvalue2\033[0m\n" +
		"   0.001 \033[0;35mWARN \033[0m warn message         field1=\033[0;35mvalue1\033[0m field2=\033[0;35mvalue2\033[0m\n" +
		"   0.001 \033[0;31mERR! \033[0m error message        field1=\033[0;31mvalue1\033[0m field2=\033[0;31mvalue2\033[0m\n"
	require.Equal(t, exp, buf.String())
}
