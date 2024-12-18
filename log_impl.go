package log

import (
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"

	apex "github.com/eluv-io/apexlog-go"
	"github.com/eluv-io/apexlog-go/handlers/discard"
	"github.com/eluv-io/apexlog-go/handlers/json"
	"github.com/eluv-io/apexlog-go/handlers/memory"
	"github.com/eluv-io/log-go/handlers/console"
	"github.com/eluv-io/log-go/handlers/raw"
	"github.com/eluv-io/log-go/handlers/text"
)

var (
	rootLog *logRoot
)

func init() {
	apex.SetHandler(json.New(os.Stdout))
	rootLog = defaultLogRoot()
}

func defaultConfig() *Config {
	return &Config{
		Level:   "info",
		Handler: "text",
	}
}

func defaultLogRoot() *logRoot {
	return newLogRoot(defaultConfig())
}

func newLogRoot(c *Config) *logRoot {
	return &logRoot{
		named:     make(map[string]*Log),
		defConfig: c,
		def:       New(c),
	}
}

func getLogRoot() *logRoot {
	return rootLog
}

func def() *Log {
	return getLogRoot().def
}

type logRoot struct {
	mutex     sync.Mutex      // mutex guarding access to the "named" map
	named     map[string]*Log // named contains all named logs
	def       *Log            // def is the default Log using apex's default Log instance
	defConfig *Config         // defConfig is the default log configuration
	metrics   Metrics         // metrics
}

func (r *logRoot) sameConfig(c *Config) bool {
	return reflect.DeepEqual(r.defConfig, c)
}

func (r *logRoot) setDefault(c *Config) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.setDefaultNoLock(c)
}

func (r *logRoot) setDefaultNoLock(c *Config) {
	if r.sameConfig(c) {
		return
	}
	r.def = New(c)
	r.defConfig = c
	updateNamedLoggers(r.def, r.named)
}

func (r *logRoot) closeLogs() {
	closeLog := func(l *Log) {
		if l.get().lumberjack != nil {
			_ = l.get().lumberjack.Close()
		}
	}
	for _, l := range r.named {
		closeLog(l)
	}
	closeLog(r.def)
}

func (r *logRoot) doLocked(fn func(r *logRoot)) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	fn(r)
}

func (r *logRoot) Get(path string) *Log {
	if path == "" {
		r.mutex.Lock()
		defer r.mutex.Unlock()
		return r.def
	}
	if path[0] != '/' {
		path = "/" + path
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	log, ok := r.named[path]
	if ok {
		return log
	}

	// create the logger hierarchy for the path

	log = r.def
	var logPath = "/"    // the path corresponding to the log instance in "log"
	conf := *r.defConfig // copy defConfig
	idx := 0
	for idx < len(path) {
		idx++ // skip the "current" separator
		if i := strings.Index(path[idx:], "/"); i != -1 {
			idx += i
		} else {
			idx = len(path)
		}
		p := path[:idx]
		l, logFound := r.named[p]
		if logFound {
			log = l
			logPath = p
			conf = *l.get().config
		}
		if c, configFound := r.defConfig.Named[p]; configFound {
			if !logFound {
				// there is a config at this level, but no log yet.
				// copy the merged configuration and create a new log from it
				mergeConfig(c, &conf)
				cc := conf
				log = newLog(&cc, defaultFields(&cc, p), log)
				r.named[p] = log
				logPath = p
			}
		}
	}
	if logPath == path {
		return log
	}

	cc := conf
	log = newLog(&cc, defaultFields(&cc, path), log)
	r.named[path] = log
	return log
}

// updates the currently available named loggers according to the new 'root'
// configuration.
func updateNamedLoggers(root *Log, named map[string]*Log) {

	for _, path := range sortedKeys(named) {
		log := named[path]
		rootConfig := root.get().config
		conf := *(rootConfig) // copy defConfig
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
			if cfg, found := rootConfig.Named[p]; found {
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
		log.updateFrom(nl)
	}
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
	var par *logger
	if parent != nil {
		par = parent.get()
	}

	if par != nil && par.config.Handler == c.Handler && reflect.DeepEqual(par.config.File, file) {
		// re-use the parent's handler if of same type
		handler = par.logger().Handler
	} else {
		metrics().InstanceCreated()
		if file != nil {
			ljack = NewLumberjackLogger(file)
			writer = ljack
			metrics().FileCreated()
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

	apexLogger := &apex.Logger{
		Handler: handler,
		Level:   level,
	}
	name := ""
	var log apex.Interface = apexLogger
	if fields != nil {
		log = apexLogger.WithFields(fields)
		name, _ = fields.Get("logger").(string)
	}
	ret := &Log{}
	ret.lw.Store(&logger{
		log:        log,
		name:       name,
		config:     c,
		lumberjack: ljack,
	})
	return ret
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
	if c.Caller != nil {
		target.Caller = c.Caller
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
