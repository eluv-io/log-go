package log_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/apexlog-go/handlers/memory"
	"github.com/eluv-io/log-go"
)

func TestLoggingToFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "logtest")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()
	f := filepath.Join(dir, "test.log")
	c := &log.Config{
		Level:   "debug",
		Handler: "json",
		File: &log.LumberjackConfig{
			Filename:   f,
			MaxSize:    10,
			MaxAge:     0,
			MaxBackups: 1,
			LocalTime:  false,
			Compress:   true,
		},
	}

	jsn, err := json.MarshalIndent(c, "", "  ")
	assert.NoError(t, err)
	fmt.Println(string(jsn))
	c2 := &log.Config{}
	err = json.Unmarshal(jsn, c2)
	assert.NoError(t, err)

	l := log.New(c2)
	l.Debug("test log message")

	finfo, err := os.Stat(f)
	assert.NoError(t, err)
	assert.Equal(t, "test.log", finfo.Name())

	file, err := os.Open(f)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	bb := make([]byte, finfo.Size())
	n, err := file.Read(bb)
	require.NoError(t, err)
	lf := make(map[string]interface{})
	_ = json.Unmarshal(bb[0:n], &lf)
	require.NotEmpty(t, lf["fields"])
}

func TestLoggingToConsole(t *testing.T) {
	dir, err := os.MkdirTemp("", "logtest")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()

	c := &log.Config{
		Level:   "debug",
		Handler: "console",
	}
	jsn, err := json.MarshalIndent(c, "", "  ")
	assert.NoError(t, err)
	fmt.Println(string(jsn))
	c2 := &log.Config{}
	err = json.Unmarshal(jsn, c2)
	assert.NoError(t, err)

	fname := filepath.Join(dir, "stdout")
	//fmt.Println("stdout is now set to", fname)
	old := os.Stdout              // keep backup of the real stdout
	temp, err := os.Create(fname) // create temp file
	require.NoError(t, err)
	os.Stdout = temp

	l := log.New(c2)
	l.Debug("test log message")

	_ = temp.Close()
	os.Stdout = old

	finfo, err := os.Stat(fname)
	assert.NoError(t, err)
	file, err := os.Open(fname)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	bb := make([]byte, finfo.Size())
	n, err := file.Read(bb)
	require.NoError(t, err)
	s := string(bb[0:n])
	fmt.Println(s)
	require.NotEmpty(t, s)
	require.False(t, strings.Contains(s, "fields"))
	require.False(t, strings.Contains(s, "logger"))
}

func TestAll(t *testing.T) {
	logger := log.New(
		&log.Config{
			Handler: "memory",
			Level:   "debug",
		})
	handler := logger.Handler().(*memory.Handler)
	doTest(t, handler, logger.Debug)
	doTest(t, handler, logger.Info)
	doTest(t, handler, logger.Warn)
	doTest(t, handler, logger.Error)
	// can't test fatal, since it calls os.Exit() ...
	// doTest(t, log.Fatal)
}

type Address struct {
	Name    string
	Street  string
	Zip     int
	City    string
	Country string
}

var address = Address{Name: "Me", Street: "Sesame Street 1", Zip: 99999, City: "Frogville", Country: "Outer Space"}

func doTest(t *testing.T, handler *memory.Handler, f func(msg string, fields ...interface{})) {
	handler.Entries = nil // clear previous entries

	f("simple message")
	assert.Equal(t, "simple message", handler.Entries[0].Message)
	assert.Equal(t, 1, len(handler.Entries[0].Fields))
	assert.Equal(t, "/", handler.Entries[0].Fields.Get("logger"))
	handler.Entries = nil // clear previous entries

	f("message with field", "user", "me")
	assert.Equal(t, "message with field", handler.Entries[0].Message)
	assert.Equal(t, 2, len(handler.Entries[0].Fields))
	assert.Equal(t, "/", handler.Entries[0].Fields.Get("logger"))
	assert.Equal(t, "me", handler.Entries[0].Fields.Get("user"))
	handler.Entries = nil // clear previous entries

	f("message with two fields", "user", "me", "age", 24)
	assert.Equal(t, "message with two fields", handler.Entries[0].Message)
	assert.Equal(t, 3, len(handler.Entries[0].Fields))
	assert.Equal(t, "/", handler.Entries[0].Fields.Get("logger"))
	assert.Equal(t, "me", handler.Entries[0].Fields.Get("user"))
	assert.Equal(t, 24, handler.Entries[0].Fields.Get("age"))
	handler.Entries = nil // clear previous entries

	f("message with incomplete fields", "user", "me", address)
	assert.Equal(t, "message with incomplete fields", handler.Entries[0].Message)
	assert.Equal(t, 3, len(handler.Entries[0].Fields))
	assert.Equal(t, "/", handler.Entries[0].Fields.Get("logger"))
	assert.Equal(t, "me", handler.Entries[0].Fields.Get("user"))
	assert.Equal(t, address, handler.Entries[0].Fields.Get("unknown"))
	handler.Entries = nil // clear previous entries

	f("non-string key (converted by log wrapper)", address, "address")
	assert.Equal(t, 2, len(handler.Entries[0].Fields))
	handler.Entries = nil // clear previous entries

	fields := []interface{}{"user", "me", "age", 24}
	f("message with two fields passed as slice", fields...)
	assertEntries(t, handler, "message with two fields passed as slice", fields)
	handler.Entries = nil // clear previous entries

	f("message with two fields passed as slice (forgetting the ellipsis)", fields)
	assertEntries(t, handler, "message with two fields passed as slice (forgetting the ellipsis)", fields)
	handler.Entries = nil // clear previous entries

}
func assertEntries(t *testing.T, handler *memory.Handler, msg string, fields []interface{}) {
	assert.Equal(t, msg, handler.Entries[0].Message)
	assert.Equal(t, len(fields)/2+1, len(handler.Entries[0].Fields))
	assert.Equal(t, "/", handler.Entries[0].Fields.Get("logger"))
	for i := 0; i+1 < len(fields); i += 2 {
		fmt.Printf("field [%s]\n", fields[i])
		assert.Equal(t, fields[i+1], handler.Entries[0].Fields.Get(fields[i].(string)))
	}
}

func TestUpdateLevelSetDefault(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "TestUpdateLevelSetDefault")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()

	c := newLogConfigDir(false, dir)
	log.SetDefault(c)
	llog := log.Get("/http-req")
	require.False(t, llog.IsDebug())
	llog.Info("this is info 1")
	badDebug := "this is debug bad"
	llog.Debug(badDebug)

	c = newLogConfigDir(true, dir)
	log.SetDefault(c)
	require.True(t, llog.IsDebug())
	llog.Info("this is info 2")
	llog.Debug("this is debug ok")
	badTrace := "this is trace bad"
	llog.Trace(badTrace)

	bb, err := os.ReadFile(filepath.Join(dir, "qfab-http-req.log"))
	require.NoError(t, err)
	sc := bufio.NewScanner(bytes.NewReader(bb))
	found := map[string]bool{
		"this is info 1":   false,
		"this is info 2":   false,
		"this is debug ok": false,
	}
	for lineNum := 0; sc.Scan(); lineNum++ {
		l := sc.Text()
		for k := range found {
			if strings.Contains(l, k) {
				found[k] = true
			}
			require.False(t, strings.Contains(l, badDebug))
			require.False(t, strings.Contains(l, badTrace))
		}
	}
	for k, ok := range found {
		require.True(t, ok, "not found %s", k)
	}

}

func newLogConfigDir(debug bool, dir string) *log.Config {
	c := log.NewConfig()
	c.File = &log.LumberjackConfig{
		Filename: filepath.Join(dir, "qfab.log"),
	}
	c.Named = make(map[string]*log.Config)
	{
		statsLog := log.NewConfig()
		statsLog.Handler = "text"
		statsLog.File = &log.LumberjackConfig{
			Filename: filepath.Join(dir, "qfab-http-stats.log"),
		}
		c.Named["/http-stats"] = statsLog
	}
	{
		reqLog := log.NewConfig()
		if debug {
			reqLog.Level = "debug"
		}
		reqLog.Handler = "raw"
		reqLog.File = &log.LumberjackConfig{
			Filename: filepath.Join(dir, "qfab-http-req.log"),
		}
		c.Named["/http-req"] = reqLog
	}
	return c
}

func TestSetLevel(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "TestSetLevel")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()

	c := newLogConfigDir(false, dir)
	log.SetDefault(c)
	llog := log.Get("/http-req")
	require.False(t, llog.IsDebug())
	llog.Info("this is info 1")
	bad := "this is debug bad"
	llog.Debug(bad)

	llog.SetDebug()
	require.True(t, llog.IsDebug())
	llog.Info("this is info 2")
	llog.Debug("this is debug ok")

	bb, err := os.ReadFile(filepath.Join(dir, "qfab-http-req.log"))
	require.NoError(t, err)
	sc := bufio.NewScanner(bytes.NewReader(bb))
	found := map[string]bool{
		"this is info 1":   false,
		"this is info 2":   false,
		"this is debug ok": false,
	}
	for lineNum := 0; sc.Scan(); lineNum++ {
		l := sc.Text()
		for k := range found {
			if strings.Contains(l, k) {
				found[k] = true
			}
			require.False(t, strings.Contains(l, bad))
		}
	}
	for k, ok := range found {
		require.True(t, ok, "not found %s", k)
	}

}

// TestConcurrent is meant to be run with -race and output no race
func TestConcurrent(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "TestConcurrent")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(dir) }()

	c := newLogConfigDir(false, dir)
	log.SetDefault(c)

	do := func(debug bool, logPath string) {
		c := newLogConfigDir(debug, dir)
		log.SetDefault(c)
		for i := 0; i < 10; i++ {
			llog := log.Get(logPath)
			llog.Info("this is info 1")
			llog.Debug("this is debug ")
		}
	}
	logPaths := []string{
		"/http-req",
		"/http",
	}

	wg := sync.WaitGroup{}
	count := 5
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()
			do(i%2 == 0, logPaths[i%2])
		}(i)
	}
	wg.Wait()
}
