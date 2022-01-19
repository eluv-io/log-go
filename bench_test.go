package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	apex "github.com/apex/log"
	"github.com/stretchr/testify/require"
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
	bb, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewReader(bb)

	ret := []*entry{}
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
	path, err := ioutil.TempDir(os.TempDir(), "TestSimpleLog")
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

func BenchmarkLog(b *testing.B) {
	path, err := ioutil.TempDir(os.TempDir(), "benchmarkLog")
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
		max := 10
		if max >= len(entries) {
			max = len(entries) - 1
		}
		for i := 0; i < max; i++ {
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
