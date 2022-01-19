// Package log provides a minimal wrapper around an arbitrary third-party
// logging library.
//
// The wrapper abstracts from a concrete logging implementation in order to
// allow painless and seamless switch-over to alternative implementations in
// the future. It provides the following functionality:
//
// - Levelled logging through the Debug, Info, Warn, Error and Fatal log methods.
//
// - Structured logging using named fields (i.e. key-value pairs) for
// simplified log parsing. Depending on the underlying log implementation,
// log entries can be written in json format, simplifying log parsing even
// further.
//
// - Super-simple function signatures. All log functions take a message string
// and an arbitrary number of additional arguments interpreted as fields
// (key-value pairs) for the structured logging.
//
// The log message should be static (i.e. it should not contain any dynamic
// content), so that log parsing remains simple. Any dynamic data should be
// added as additional fields.
//
// Example:
//     // conventional plain-text log
//     log.Infof("uploading %s for %s", filename, user)
//     log.Warnf("upload failed %s for %s: %s", filename, user, err)
//
//     // structured logging
//    log.Info("uploading", "file", filename, "user", user)
//    log.Warn("upload failed", "file", filename, "user", user, "error", err)
//
//    // same as above: an error can be logged without specifying a key ("error" will be used as key)
//    log.Warn("upload failed", "file", filename, "user", user, err)
//
// Some background on structured logging: https://medium.com/@tjholowaychuk/apex-log-e8d9627f4a9a
package log

import (
	"io"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"

	apex "github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/apexlog-go/handlers/discard"
	"github.com/eluv-io/apexlog-go/handlers/json"
	"github.com/eluv-io/apexlog-go/handlers/memory"
	"github.com/eluv-io/log-go/handlers/console"
	"github.com/eluv-io/log-go/handlers/raw"
	"github.com/eluv-io/log-go/handlers/text"
	"github.com/modern-go/gls"
	"gopkg.in/natefinch/lumberjack.v2"
)

// defConfig is the default log configuration
var defConfig = &Config{
	Level:   "info",
	Handler: "text",
}

// def is the default Log using apex's default Log instance
var def = newLog(defConfig, defaultFields(defConfig, "/"), nil)

// named contains all named logs
var named = make(map[string]*Log)

// mutex guarding access to the "named" map
var mutex sync.Mutex

func init() {
	apex.SetHandler(json.New(os.Stdout))
}

// SetDefault sets the default configuration and creates the default log based on that configuration.
func SetDefault(c *Config) {
	if reflect.DeepEqual(defConfig, c) {
		return
	}
	def = New(c)
	defConfig = c
	updateNamedLoggers(def)
}

func defaultFields(c *Config, path string) *apex.Fields {
	switch c.Handler {
	case "console":
		return &apex.Fields{}
	case "memory":
		if c.Level != "debug" {
			return &apex.Fields{}
		}
	}

	return &apex.Fields{{Name: "logger", Value: path}}
}

// New creates a new root Logger
func New(c *Config) *Log {
	return newLog(c, defaultFields(c, "/"), nil)
}

// newLog creates a new Log wrapper from the given configuration and additional
// log fields
func newLog(c *Config, fields *apex.Fields, parent *Log) *Log {
	var ljack *lumberjack.Logger
	var writer io.Writer = os.Stdout

	level, err := apex.ParseLevel(c.Level)
	if err != nil {
		level = apex.InfoLevel
	}

	file := c.File
	if file != nil && file.Filename == "" {
		// no filename is equivalent to logging to stdout
		file = nil
	}

	var handler apex.Handler

	if parent != nil && parent.config.Handler == c.Handler && reflect.DeepEqual(parent.config.File, file) {
		// re-use the parent's handler if of same type
		handler = parent.handler
	} else {
		metrics.InstanceCreated()
		if file != nil {
			ljack = NewLumberjackLogger(file)
			writer = ljack
			metrics.FileCreated()
		}
		switch c.Handler {
		case "text":
			handler = text.New(writer)
		case "raw":
			handler = raw.New(writer)
		case "console":
			handler = console.New(writer)
		case "discard":
			handler = discard.Default
		case "memory":
			handler = memory.New()
		case "json":
			fallthrough
		default:
			handler = json.New(writer)
		}
	}

	logger := &apex.Logger{
		Handler: handler,
		Level:   level,
	}
	name := ""
	var log apex.Interface = logger
	if fields != nil {
		log = logger.WithFields(fields)
		name, _ = fields.Get("logger").(string)
	}
	return &Log{
		logger:     logger,
		log:        log,
		name:       name,
		config:     c,
		handler:    handler,
		lumberjack: ljack,
	}
}

func NewLumberjackLogger(c *LumberjackConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   c.Filename,
		MaxSize:    c.MaxSize,
		MaxAge:     c.MaxAge,
		MaxBackups: c.MaxBackups,
		LocalTime:  c.LocalTime,
		Compress:   c.Compress,
	}
}

// Get returns the named logger for the given path. Loggers are organized in a
// hierarchy (tree) defined by their paths. Paths use a forward slash '/' as
// separator (e.g. /eluvio/util/json). Loggers inherit attributes
// of their parent loggers.
// A logger's path is added to every log entry as a field: logger=/eluvio/util/json
func Get(path string) *Log {
	if path == "" {
		return def
	}
	if path[0] != '/' {
		path = "/" + path
	}

	mutex.Lock()
	defer mutex.Unlock()

	log, ok := named[path]
	if ok {
		return log
	}

	// create the logger hierarchy for the path

	log = def
	var logPath = "/"  // the path corresponding to the log instance in "log"
	conf := *defConfig // copy defConfig
	idx := 0
	for idx < len(path) {
		idx++ // skip the "current" separator
		if i := strings.Index(path[idx:], "/"); i != -1 {
			idx += i
		} else {
			idx = len(path)
		}
		p := path[:idx]
		l, logFound := named[p]
		if logFound {
			log = l
			logPath = p
		}
		if c, configFound := defConfig.Named[p]; configFound {
			mergeConfig(c, &conf)
			if !logFound {
				// there is a config at this level, but no log yet.
				// copy the merged configuration and create a new log from it
				cc := conf
				log = newLog(&cc, defaultFields(&cc, p), log)
				named[p] = log
				logPath = p
			}
		}
	}
	if logPath == path {
		return log
	}

	cc := conf
	log = newLog(&cc, defaultFields(&cc, path), log)
	named[path] = log
	return log
}

// Root retrieves the root logger - same as Get("/")
func Root() *Log {
	return Get("/")
}

// mergeConfig merges the given config c into the target config.
func mergeConfig(c *Config, target *Config) {
	if c.Level != "" {
		target.Level = c.Level
	}
	if c.Handler != "" {
		target.Handler = c.Handler
	}
	if c.File != nil {
		target.File = c.File
	}
	if c.GoRoutineID != nil {
		b := *c.GoRoutineID
		target.GoRoutineID = &b
	}
}

// updates the currently available named loggers according to the new
// configuration.
func updateNamedLoggers(root *Log) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, path := range sortedKeys(named) {
		log := named[path]
		conf := *(root.config) // copy defConfig
		parent := root
		idx := 0
		for idx < len(path) {
			idx++ // skip the "current" separator
			if i := strings.Index(path[idx:], "/"); i != -1 {
				idx += i
			} else {
				idx = len(path)
			}
			p := path[:idx]
			if cfg, found := root.config.Named[p]; found {
				mergeConfig(cfg, &conf)
			}
			if p != path {
				if l, found := named[p]; found {
					parent = l
				}
			}
		}
		nl := newLog(&conf, defaultFields(&conf, path), parent)
		// replace all members of current log instance with newly created ones
		log.config = nl.config
		log.handler = nl.handler
		log.log = nl.log
		log.logger = nl.logger
		log.name = nl.name
		log.lumberjack = nl.lumberjack
	}
}

func sortedKeys(m map[string]*Log) []string {
	keys := make([]string, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}

// Log is the logger wrapping an apex Log instance
type Log struct {
	// logger is the underlying apex.Logger of log (apex.Interface) - needed for
	// controlling the log level
	logger *apex.Logger
	// log is the logger decorated with the logger name field
	log apex.Interface
	// name is the logger's name when created through Get()
	name       string
	config     *Config
	handler    apex.Handler
	lumberjack *lumberjack.Logger
}

func (l *Log) Handler() apex.Handler {
	return l.handler
}

// Trace logs the given message at the Trace level.
func Trace(msg string, fields ...interface{}) {
	def.Trace(msg, fields...)
}

// Trace logs the given message at the Trace level.
func (l *Log) Trace(msg string, fields ...interface{}) {
	metrics.Debug(l.name)
	if l.IsTrace() {
		l.log.Trace(msg, l.fields(fields)...)
	}
}

// Debug logs the given message at the Debug level.
func Debug(msg string, fields ...interface{}) {
	def.Debug(msg, fields...)
}

// Debug logs the given message at the Debug level.
func (l *Log) Debug(msg string, fields ...interface{}) {
	metrics.Debug(l.name)
	if l.IsDebug() {
		l.log.Debug(msg, l.fields(fields)...)
	}
}

// Info logs the given message at the Info level.
func Info(msg string, fields ...interface{}) {
	def.Info(msg, fields...)
}

// Info logs the given message at the Info level.
func (l *Log) Info(msg string, fields ...interface{}) {
	metrics.Info(l.name)
	if l.IsInfo() {
		l.log.Info(msg, l.fields(fields)...)
	}
}

// Warn logs the given message at the Warn level.
func Warn(msg string, fields ...interface{}) {
	def.Warn(msg, fields...)
}

// Warn logs the given message at the Warn level.
func (l *Log) Warn(msg string, fields ...interface{}) {
	metrics.Warn(l.name)
	if l.IsWarn() {
		l.log.Warn(msg, l.fields(fields)...)
	}
}

// Error logs the given message at the Error level.
func Error(msg string, fields ...interface{}) {
	def.Error(msg, fields...)
}

// Error logs the given message at the Error level.
func (l *Log) Error(msg string, fields ...interface{}) {
	metrics.Error(l.name)
	if l.IsError() {
		l.log.Error(msg, l.fields(fields)...)
	}
}

// Fatal logs the given message at the Fatal level.
func Fatal(msg string, fields ...interface{}) {
	def.Fatal(msg, fields...)
}

// Fatal logs the given message at the Fatal level.
func (l *Log) Fatal(msg string, fields ...interface{}) {
	l.log.Fatal(msg, l.fields(fields)...)
}

// IsTrace returns true if the logger logs in Trace level.
func IsTrace() bool {
	return def.logger.Level <= apex.TraceLevel
}

// IsTrace returns true if the logger logs in Trace level.
func (l *Log) IsTrace() bool {
	return l.logger.Level <= apex.TraceLevel
}

// IsDebug returns true if the logger logs in Debug level.
func IsDebug() bool {
	return def.logger.Level <= apex.DebugLevel
}

// IsDebug returns true if the logger logs in Debug level.
func (l *Log) IsDebug() bool {
	return l.logger.Level <= apex.DebugLevel
}

// IsInfo returns true if the logger logs in Info level.
func IsInfo() bool {
	return def.logger.Level <= apex.InfoLevel
}

// IsInfo returns true if the logger logs in Info level.
func (l *Log) IsInfo() bool {
	return l.logger.Level <= apex.InfoLevel
}

// IsWarn returns true if the logger logs in Warn level.
func IsWarn() bool {
	return def.logger.Level <= apex.WarnLevel
}

// IsWarn returns true if the logger logs in Warn level.
func (l *Log) IsWarn() bool {
	return l.logger.Level <= apex.WarnLevel
}

// IsError returns true if the logger logs in Error level.
func IsError() bool {
	return def.logger.Level <= apex.ErrorLevel
}

// IsError returns true if the logger logs in Error level.
func (l *Log) IsError() bool {
	return l.logger.Level <= apex.ErrorLevel
}

// IsFatal returns true if the logger logs in Fatal level.
func IsFatal() bool {
	return def.logger.Level <= apex.FatalLevel
}

// IsFatal returns true if the logger logs in Fatal level.
func (l *Log) IsFatal() bool {
	return l.logger.Level <= apex.FatalLevel
}

// Name returns the name of this logger
func (l *Log) Name() string {
	return l.name
}

func (l *Log) Level() string {
	return l.logger.Level.String()
}

// SetLevel sets the log level according to the given string.
func (l *Log) SetLevel(level string) {
	lvl, err := apex.ParseLevel(level)
	if err != nil {
		return
	}
	l.setLogLevel(lvl)
}

// SetDebug sets the log level to Debug.
func (l *Log) SetDebug() {
	l.setLogLevel(apex.DebugLevel)
}

// SetInfo sets the log level to Info.
func (l *Log) SetInfo() {
	l.setLogLevel(apex.InfoLevel)
}

// SetWarn sets the log level to Warn.
func (l *Log) SetWarn() {
	l.setLogLevel(apex.WarnLevel)
}

// SetError sets the log level to Error.
func (l *Log) SetError() {
	l.setLogLevel(apex.ErrorLevel)
}

// SetFatal sets the log level to Fatal.
func (l *Log) SetFatal() {
	l.setLogLevel(apex.FatalLevel)
}

func (l *Log) setLogLevel(level apex.Level) {
	for name, logger := range named {
		if strings.HasPrefix(name, l.Name()) {
			logger.logger.Level = level
		}
	}
	l.logger.Level = level
}

func (l *Log) fields(args []interface{}) []interface{} {
	if l.config.GoRoutineID != nil && *l.config.GoRoutineID {
		args = append(args, "gid", goID())
	}
	return args
}

// Call invokes the function and simply logs if an error occurs. Useful when
// deferring a call like io.Close:
//   defer log.Call(reader.Close, "package.Func")
//
// Optionally provide a log function:
//   log.Call(reader.Close, "package.Func", mylog.Debug)
func (l *Log) Call(f func() error, msg string, log ...func(msg string, fields ...interface{})) {
	if f != nil {
		if err := f(); err != nil {
			fname := "unknown"
			if ffp := runtime.FuncForPC(reflect.ValueOf(f).Pointer()); ffp != nil {
				fname = ffp.Name()
			}
			if len(log) > 0 {
				log[0](msg, err, "func", fname)
			} else {
				l.Info(msg, err, "func", fname)
			}
		}
	}
}

func CloseLogFiles() {
	closeLog := func(l *Log) {
		if l.lumberjack != nil {
			_ = l.lumberjack.Close()
		}
	}
	for _, l := range named {
		closeLog(l)
	}
	closeLog(def)
}

// goID returns the goroutine id of current goroutine
func goID() int64 {
	return gls.GoID()
}
