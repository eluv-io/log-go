package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	apex "github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/errors-go"
)

type entry struct {
	Message string
	Fields  []interface{}
}

func newEntry(e *apex.Entry, maxFields int) *entry {
	var fields []interface{}
	count := 0
	for k, v := range e.Fields.Map() {
		fields = append(fields, k, v)
		count++
		if maxFields > 0 && count >= maxFields {
			break
		}
	}
	return &entry{
		Message: e.Message,
		Fields:  fields,
	}
}

func testLogs(maxFields int) ([]*entry, error) {
	f := "testdata/anon_usage.log"
	bb, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewReader(bb)

	ret := make([]*entry, 0)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		ln := scanner.Bytes()
		if len(ln) == 0 {
			continue
		}
		e := &apex.Entry{}
		err = json.Unmarshal(ln, e)
		if err != nil {
			Warn("invalid log", err, "log", string(ln))
			continue
		}
		ret = append(ret, newEntry(e, maxFields))
	}
	return ret, nil
}

func TestSimpleLog(b *testing.T) {
	path, err := os.MkdirTemp(os.TempDir(), "TestSimpleLog")
	require.NoError(b, err)
	defer func() { _ = os.RemoveAll(path) }()

	entries, err := testLogs(10)
	require.NoError(b, err)

	cfg := &Config{
		Level:   "info",
		Handler: "text",
		File: &LumberjackConfig{
			Filename: filepath.Join(path, "f.log"),
		}}
	log := newLog(cfg, defaultFields(cfg, "/"), nil)

	e := entries[200%len(entries)]
	log.Info(e.Message, e.Fields...)
}

// -- before refactor --
//BenchmarkLog/file-config-8         	   50097	     22658 ns/op	   10931 B/op	      49 allocs/op
//BenchmarkLog/file-config-8         	   44073	     23697 ns/op	    9084 B/op	      48 allocs/op
//
// -- after refactor --
//BenchmarkLog/file-config-8         	   80040	     13850 ns/op	    5796 B/op	      42 allocs/op
//BenchmarkLog/file-config-8         	   86277	     14392 ns/op	    4537 B/op	      42 allocs/op
//
// limited to 10 fields (for comparison with https://github.com/uber-go/zap#performance)
//BenchmarkLog/file-config-8         	   95760	     12208 ns/op	    3914 B/op	      51 allocs/op
//BenchmarkLog/file-config-10-fields-8     156030	      7501 ns/op	    1153 B/op	      24 allocs/op
//
//BenchmarkLog/file-config-8         	   93165	     12167 ns/op	    3888 B/op	      51 allocs/op
//BenchmarkLog/file-config-10-fields-8     180730	      7061 ns/op	    1153 B/op	      24 allocs/op
//
// -- jan 2024 --
// commit: master@017ba5a5be4d5227b9f025cff995beab928d7b75
//BenchmarkLog/file-config-8         	   71143	     17048 ns/op	    3767 B/op	      51 allocs/op
//BenchmarkLog/file-config-10-fields-8     125799	      9259 ns/op	    1153 B/op	      24 allocs/op
//
// -- with no mod
//BenchmarkLog/file-config-8         	   71025	     17209 ns/op	    3789 B/op	      51 allocs/op
//BenchmarkLog/file-config-10-fields-8     125124	      9562 ns/op	    1153 B/op	      24 allocs/op
// -- with atomic wrapper
//BenchmarkLog/file-config-8         	   69007	     17866 ns/op	    3798 B/op	      51 allocs/op
//BenchmarkLog/file-config-10-fields-8     122730	     10282 ns/op	    1153 B/op	      24 allocs/op
// -- with atomic wrapper (2)
//BenchmarkLog/file-config-8         	   72332	     16735 ns/op	    3793 B/op	      51 allocs/op
//BenchmarkLog/file-config-10-fields-8     135945	      8894 ns/op	    1153 B/op	      24 allocs/op

func BenchmarkLog(b *testing.B) {
	path, err := os.MkdirTemp(os.TempDir(), "benchmarkLog")
	require.NoError(b, err)
	defer func() { _ = os.RemoveAll(path) }()

	entries, err := testLogs(10)
	require.NoError(b, err)
	require.Equal(b, 1000, len(entries))

	{
		cfg := &Config{
			Level:   "info",
			Handler: "text",
		}
		log := newLog(cfg, defaultFields(cfg, "/"), nil)
		maxent := 10
		if maxent >= len(entries) {
			maxent = len(entries) - 1
		}
		for i := 0; i < maxent; i++ {
			e := entries[i]
			log.Info(e.Message, e.Fields...)
		}
	}

	cfg := &Config{
		Level:   "info",
		Handler: "text",
		File: &LumberjackConfig{
			Filename: filepath.Join(path, "f.log"),
		}}
	log := newLog(cfg, defaultFields(cfg, "/"), nil)
	b.Run("file-config", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			e := entries[i%len(entries)]
			log.Info(e.Message, e.Fields...)
		}
	})

	// test with pre-allocated fields
	defaultFields := func(c *Config, path string) *apex.Fields {
		f := apex.Fields(nil).
			Append("logger", path).
			Append("name", "me").
			Append("count", 1).
			Append("age", 444).
			Append("location", "here").
			Append("town", "valencia").
			Append("country", "spain").
			Append("planet", "earth").
			Append("more_count", 444).
			Append("other_location", "there")
		return &f
	}
	log = newLog(cfg, defaultFields(cfg, "/"), nil)
	b.Run("file-config-10-fields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			e := entries[i%len(entries)]
			log.Info(e.Message)
		}
	})

}

// BenchmarkNoLog tests logging with a logger configured to info:
// - at debug level in order to benchmark 'disabled' logs
// - at info with a discard logger
//
// == before changes using atomic
// master@017ba5a5be4d5227b9f025cff995beab928d7b75
// goos: darwin - goarch: amd64
// BenchmarkNoLog/debug-10-fields-8         	 8659641	       126.7 ns/op	     320 B/op	       1 allocs/op
// BenchmarkNoLog/discard-10-fields-8       	  271183	      4302 ns/op	    1849 B/op	      31 allocs/op
// == after changes using atomic
// BenchmarkNoLog/debug-10-fields-8         	 9258938	       130.9 ns/op	     320 B/op	       1 allocs/op
// BenchmarkNoLog/discard-10-fields-8       	  274232	      4264 ns/op	    1849 B/op	      31 allocs/op
func BenchmarkNoLog(b *testing.B) {
	doLog := func(l *Log, level apex.Level, msg string) {
		switch level {
		case apex.InfoLevel:
			l.Info(msg,
				"name", "me",
				"count", 1,
				"age", 444,
				"location", "here",
				"town", "valencia",
				"country", "spain",
				"planet", "earth",
				"more_count", 444,
				"other_location", "there",
				"more_location", "more loc",
			)
		case apex.DebugLevel:
			l.Debug(msg,
				"name", "me",
				"count", 1,
				"age", 444,
				"location", "here",
				"town", "valencia",
				"country", "spain",
				"planet", "earth",
				"more_count", 444,
				"other_location", "there",
				"more_location", "more loc",
			)
		default:
			panic(errors.Str("unhandled level"))
		}
	}
	logDebug := func(l *Log, msg string) {
		doLog(l, apex.DebugLevel, msg)
	}
	logInfo := func(l *Log, msg string) {
		doLog(l, apex.InfoLevel, msg)
	}

	log := New(&Config{
		Level:   "info",
		Handler: "text",
	})
	b.Run("debug-10-fields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logDebug(log, "hi")
		}
	})

	log = New(&Config{
		Level:   "info",
		Handler: "discard",
	})
	b.Run("discard-10-fields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logInfo(log, "hi")
		}
	})

}
