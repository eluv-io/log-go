package main

import (
	"io"

	"github.com/eluv-io/errors-go"
	elog "github.com/eluv-io/log-go"
	_ "github.com/eluv-io/log-go/sample/config"
	"github.com/eluv-io/log-go/sample/sub"
)

// log contains the package-local log instance. Every package should define its
// own instance, retrieved with the full package name. Define the variable in
// the package's principle go source file.
//
// The package name serves two purposes:
// 1. It is added to each log entry as a field called "logger". This allows to
//    determine easily which part of the code (package) has generated the log
//    entry.
// 2. Logging can be configured separately for each named log instance, i.e.
//    for each package if this convention is followed strictly.
var log = elog.Get("/eluvio/log/sample")

func createAccount(ID string, name string) (interface{}, error) {

	// Structured logging:
	// - the text message is short and describes the operation or failure
	// - all non-static information is logged as fields, with a field name and
	//   the value.
	// - field names are self-describing and follow json naming conventions:
	//   all lower case with underscores.
	//
	// In addition, log levels allow to mark the importance of the log message.
	// Levels are: trace, debug, info, warn, error, fatal. Logging as fatal
	// immediately stops execution of the process with a call to os.Exit(1).
	log.Debug("creating account", "account_id", ID, "account_name", name)

	if ID == "123456" {
		// Errors are created with the eluvio/errors package.
		err := io.EOF
		return nil, errors.E("create account", errors.K.Exist, err, "account_id", ID)
	}

	log.Info("account created", "account_id", ID, "account_name", name)
	log.Trace("createAccount finished")

	return ID, nil
}

func main() {
	log.Info("logging example")

	_, err := createAccount("123456", "Test Account")
	if err != nil {
		// Errors are logged in the location where they are handled. Always
		// include the original error. Errors can be logged without
		// field name - they are automatically assigned to the "error" key.
		log.Warn("failed to create account", err)

		_, err = createAccount("456789", "Another Test Account")
	}

	// logging can be configured separately for each package
	// -> see conf/log_config_sample.go
	//
	// anything logged by the "sub" package in this example uses the json
	// formatter and the INFO log level.
	sub.Call("arg1", "arg2", "arg3")

	log.Fatal("reached the end")
}

// The sample produces the following log output:
//   0.000       logging example
//   0.000 DBG   creating account     account_id=123456 account_name=Test Account
//   0.000 WARN  failed to create account error=op [create account] kind [item already exists] account_id [123456] cause [EOF]
//        sample/log_sample.go:41 createAccount()
//        sample/log_sample.go:52 main()
//
//   0.000 DBG   creating account     account_id=456789 account_name=Another Test Account
//   0.000       account created      account_id=456789 account_name=Another Test Account
//{"fields":{"logger":"/eluvio/log/sample/sub","args":["arg1","arg2","arg3"]},"level":"info","timestamp":"2022-01-12T15:35:44.854894+01:00","message":"call to sub"}
//{"fields":{"logger":"/eluvio/log/sample/sub","error":{"cause":"EOF","file":"/tmp/app-config.yaml","kind":"I/O error","op":"failed to parse config","stacktrace":"\tgithub.com/eluv-io/log-go/sample/sub/sub.go:18 Call()\n\tsample/log_sample.go:67                                   main()\n"}},"level":"warn","timestamp":"2022-01-12T15:35:44.855069+01:00","message":"failed to read config, using defaults"}
//{"fields":{"logger":"/eluvio/log/sample/sub","error":{"cause":{"cause":"EOF","file":"/tmp/app-config.yaml","kind":"I/O error","op":"failed to parse config"},"kind":"invalid","op":"configuration incomplete: timeout missing","stacktrace":"\tgithub.com/eluv-io/log-go/sample/sub/sub.go:18 Call()\n\tgithub.com/eluv-io/log-go/sample/sub/sub.go:23 Call()\n\tsample/log_sample.go:67                                   main()\n"}},"level":"warn","timestamp":"2022-01-12T15:35:44.85513+01:00","message":"call failed"}
//   0.001 FATL  reached the end
// Process finished with exit code 1
