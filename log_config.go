package log

var (
	trueValue = true
)

// NewConfig returns a new config instance, initialized with default values
func NewConfig() *Config {
	return (&Config{}).InitDefaults()
}

type Config struct {
	// Level is the log level. Default: normal
	Level string `json:"level"`

	// Handler specifies the log handler to use. Default: json
	Handler string `json:"formatter"`

	// File specifies the log file settings. Default: nil (log to stdout)
	File *LumberjackConfig `json:"file,omitempty"`

	// Include go routine ID as 'gid' in logged fields
	GoRoutineID *bool `json:"go_routine_id,omitempty"`

	// Named contains the configuration of named loggers.
	// Any nested "Named" elements are ignored.
	Named map[string]*Config `json:"named,omitempty"`
}

func (c *Config) InitDefaults() *Config {
	c.Level = "normal"
	c.Handler = "json"
	c.GoRoutineID = &trueValue
	return c
}

// Stdout is a LumberjackConfig with an empty Filename that leads to logging to
// stdout.
var Stdout = &LumberjackConfig{}

type LumberjackConfig struct {
	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	Filename string `json:"filename"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `json:"maxsize"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `json:"maxage"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `json:"maxbackups"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"localtime"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `json:"compress"`
}
