# Logging with `eluv-io/log-go`

[![](https://github.com/eluv-io/log-go/actions/workflows/build.yaml/badge.svg)](https://github.com/eluv-io/log-go/actions?query=workflow%3Abuild)
[![CodeQL](https://github.com/eluv-io/log-go/actions/workflows/codeql-analysis.yaml/badge.svg)](https://github.com/eluv-io/log-go/actions/workflows/codeql-analysis.yaml)

The package `eluv-io/log-go` makes logging super simple, efficient, and consistent.

```go
log.Info("create account", "account_id", ID, "account_name", name)
log.Warn("failed to create account", err)
```

It is based on the following principles:

* structured logging
* levelled logging
* package-based configuration

For sample code, see

* [log_sample.go](sample/log_sample.go)
* [log_config_sample.go](sample/config/log_config_sample.go)

### Structured Logging

Pack dynamic information into discrete fields (key-value pairs), which can be used later for easy querying and subsequent processing (`eluv-io/log-go` provides a log handler that can emit log entries as JSON objects).

```go
log.Debug("creating account", "account_id", ID, "account_name", name)
log.Info("account created", "account_id", ID, "account_name", name)
log.Warn("failed to create account", err)
```

With the `text` formatter, these statements produce the following output:

```text
2018-03-02T15:23:04.317Z DEBUG creating account          account_id=123456 account_name=Test Account logger=/eluvio/log/sample
2018-03-02T15:23:04.317Z INFO  account created           account_id=456789 account_name=Another Test Account logger=/eluvio/log/sample
2018-03-02T15:23:04.317Z WARN  failed to create account  error=op [create account] kind [item already exists] account_id [123456] cause [EOF] logger=/eluvio/log/sample
```

A log call takes these arguments: a message, an optional `error`, and a list of fields (key-value pairs).

| Argument | Description                                                                                                                                                                                                                                                     |
|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| message  | The message is the first argument and should be regarded as a label for the operation or action that is being logged. Keep it short and concise - and do not include any dynamic information (which has to go into fields)!                                     |
| error    | Optional. Errors can be logged without field name - they are automatically assigned to the "error" key.                                                                                                                                                         |
| fields   | The rest of the arguments are fields with a name and a value. Field names are self-describing and follow json naming conventions: all lower case with underscores. Prefer "account_id" over just "id" in order to avoid ambiguities during log post-processing. |

Traditional logging libraries (such as golang's standard `log` package) promote the composition of string messages from all the information. This leads to inconsistent formatting, requires elaborate parsing and in general makes automatic processing cumbersome. Hence, do not use the following approach (actually, `eluv-io/log-go` does not offer such formatting methods...):

```go
// bad, do not use!
log.Infof("Creating account %s with name %s", ID, name)
log.Debugf("uploading [%s] for [%s]", filename, user)
log.Warnf("upload failed %s for %s: %s", filename, user, err)
```

### Levelled Logging

Use different log levels according to the importance of a log entry:

| Level | Description                                                                 |
|-------|-----------------------------------------------------------------------------|
| FATAL | An unrecoverable error. This stops the running process with `os.Exit(1)`!   |
| ERROR | A serious error. The application will subsequently run in a degraded state. |
| WARN  | A recoverable error, for example a failure to create a business object.     |
| INFO  | Business object lifecycle events, external events, etc.                     |
| DEBUG | Log statement helping understand the functioning of the system.             |
| TRACE | Everything else.                                                            |

Levels are mainly used to suppress log events in order to keep log size small. The default log level is INFO. Hence, do not log important information in DEBUG.

### Package-Based Configuration

Logging can be configured individually based on hierarchical names. Using the go package as name for the log instance allows per-package configuration.

In order to achieve this, each go package should define a package-local (unexported) variable called log in its main go source file:

```go
import elog "github.com/eluv-io/log-go"

var log = elog.Get("/eluvio/log/sample")
```

This creates the log instance for this package. If there is a specific configuration for the package, it is used. Otherwise, it inherits the configuration from the parent instance (i.e. from a package above or the root logger).

In other go files of the same package, no import of `eluv-io/log-go` is needed, since they will use the variable `log` instead of the package import.

Hierarchical configuration is achieved with a configuration object that is used to create the global singleton instance:

```go
config := &elog.Config{
    Level:   "debug",
    Handler: "text",
    Named: map[string]*elog.Config{
        "/eluvio/log/sample/sub": {
            Level:   "normal",
            Handler: "json",
        },
    },
}
elog.SetDefault(config)
```

This obviously only needs to be done once at application startup. The configuration struct can also be used to parse directly from JSON or YAML. 
See [the log configuration sample](sample/config/log_config_sample.go)


### Log Handlers

The following log handlers are available:

#### text

A handler for text-based log files:

```text
2018-03-02T15:23:04.317Z INFO  logging example           logger=/eluvio/log/sample
2018-03-02T15:23:04.317Z DEBUG creating account          account_id=123456 account_name=Test Account logger=/eluvio/log/sample
2018-03-02T15:23:04.317Z WARN  failed to create account  error=op [create account] kind [item already exists] account_id [123456] cause [EOF] logger=/eluvio/log/sample
2018-03-02T15:23:04.317Z DEBUG creating account          account_id=456789 account_name=Another Test Account logger=/eluvio/log/sample
2018-03-02T15:23:04.317Z INFO  account created           account_id=456789 account_name=Another Test Account logger=/eluvio/log/sample
```

#### console

A handler for output to the terminal with coloring:

```text
   0.000 INFO  logging example           logger=/eluvio/log/sample
   0.010 DEBUG creating account          account_id=123456 account_name=Test Account logger=/eluvio/log/sample
   1.425 WARN  failed to create account  error=op [create account] kind [item already exists] account_id [123456] cause [EOF] logger=/eluvio/log/sample
   1.433 DEBUG creating account          account_id=456789 account_name=Another Test Account logger=/eluvio/log/sample
   1.565 INFO  account created           account_id=456789 account_name=Another Test Account logger=/eluvio/log/sample
```

#### json

A handler emitting json objects:

```json
{"fields":{"args":["arg1","arg2","arg3"],"logger":"/eluvio/log/sample/sub"},"level":"info","timestamp":"2018-03-02T16:52:04.831614+01:00","message":"call to sub"}
{"fields":{"error":{"cause":"EOF","file":"/tmp/app-config.yaml","kind":"I/O error","op":"failed to parse config","stacktrace":"runtime/asm_amd64.s:2337: runtime.goexit:\n\truntime/proc.go:195: ...main:\n\tsample/log_sample.go:66: main.main:\n\teluvio/log/sample/sub/sub.go:17: eluvio/log/sample/sub.Call"},"logger":"/eluvio/log/sample/sub"},"level":"warn","timestamp":"2018-03-02T16:52:04.831737+01:00","message":"failed to read config, using defaults"}
{"fields":{"error":{"cause":{"cause":"EOF","file":"/tmp/app-config.yaml","kind":"I/O error","op":"failed to parse config","stacktrace":"runtime/asm_amd64.s:2337: runtime.goexit:\n\truntime/proc.go:195: ...main:\n\tsample/log_sample.go:66: main.main:\n\teluvio/log/sample/sub/sub.go:17: eluvio/log/sample/sub.Call"},"kind":"invalid","op":"configuration incomplete: timeout missing","stacktrace":"runtime/asm_amd64.s:2337: runtime.goexit:\n\truntime/proc.go:195: ...main:\n\tsample/log_sample.go:66: main.main:\n\teluvio/log/sample/sub/sub.go:22: eluvio/log/sample/sub.Call"},"logger":"/eluvio/log/sample/sub"},"level":"warn","timestamp":"2018-03-02T16:52:04.831799+01:00","message":"call failed"}
```

And pretty printed:

```json
{
  "fields": {
    "args": [
      "arg1",
      "arg2",
      "arg3"
    ],
    "logger": "/eluvio/log/sample/sub"
  },
  "level": "info",
  "timestamp": "2018-03-02T16:52:04.831614+01:00",
  "message": "call to sub"
}
{
  "fields": {
    "error": {
      "cause": "EOF",
      "file": "/tmp/app-config.yaml",
      "kind": "I/O error",
      "op": "failed to parse config",
      "stacktrace": "runtime/asm_amd64.s:2337: runtime.goexit:\n\truntime/proc.go:195: ...main:\n\tsample/log_sample.go:66: main.main:\n\teluvio/log/sample/sub/sub.go:17: eluvio/log/sample/sub.Call"
    },
    "logger": "/eluvio/log/sample/sub"
  },
  "level": "warn",
  "timestamp": "2018-03-02T16:52:04.831737+01:00",
  "message": "failed to read config, using defaults"
}
{
  "fields": {
    "error": {
      "cause": {
        "cause": "EOF",
        "file": "/tmp/app-config.yaml",
        "kind": "I/O error",
        "op": "failed to parse config",
        "stacktrace": "runtime/asm_amd64.s:2337: runtime.goexit:\n\truntime/proc.go:195: ...main:\n\tsample/log_sample.go:66: main.main:\n\teluvio/log/sample/sub/sub.go:17: eluvio/log/sample/sub.Call"
      },
      "kind": "invalid",
      "op": "configuration incomplete: timeout missing",
      "stacktrace": "runtime/asm_amd64.s:2337: runtime.goexit:\n\truntime/proc.go:195: ...main:\n\tsample/log_sample.go:66: main.main:\n\teluvio/log/sample/sub/sub.go:22: eluvio/log/sample/sub.Call"
    },
    "logger": "/eluvio/log/sample/sub"
  },
  "level": "warn",
  "timestamp": "2018-03-02T16:52:04.831799+01:00",
  "message": "call failed"
}
```

#### discard

A handler that discards all output.

### Logging to Files

In order to write logs to a file, with automatic roll-over based on size and/or time, configure it accordingly:

```json
  "log": {
    "level": "debug",
    "formatter": "text",
    "file": {
      "filename": "/var/log/qfab.log",
      "maxsize": 10,
      "maxage": 0,
      "maxbackups": 2,
      "localtime": false,
      "compress": false
    }
  },
```

File logging is implemented by the 3rd-party library [lumberjack](https://github.com/natefinch/lumberjack#type-logger). See their documentation for an explanation of all configuration parameters.

### Log Throttling

Certain situations might require throttling the output of log messages in order to prevent log pollution. This is especially true for log statements in tight loops with uncontrollable frequency, for example when the recurrence is mandated by a configuration option. Throttling can be achieved by using the `Throttle` function:

```go 
for {
	...
	log.Throttle("connect").Debug("failed to connect", "error", error, "attempt", attempt)
	...
}
```

The `Throttle` function expects a throttling key as first argument ("connect" in the example above) that is used to identify "similar" messages that may be eliminated if occurring multiple times during the throttle period. The throttle period is 5 seconds by default, but may be configured with an optional second argument:

```go 
authLog := log.Throttle("authenticate", time.Second)
for {
	...
	authLog.Debug("failed to authenticate", "error", error)
    ...
}
```

