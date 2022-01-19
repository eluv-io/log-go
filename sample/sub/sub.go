package sub

import (
	"io"

	"github.com/eluv-io/errors-go"
	elog "github.com/eluv-io/log-go"
)

var log = elog.Get("/eluvio/log/sample/sub")

func Call(args ...string) {
	log.Info("call to sub", "args", args)

	// Errors (from eluvio/errors) are serialized as json objects!
	// Hence any fields specified in the error are conserved as such
	// and accessible as nested key-value pairs of the error json object.
	err := errors.E("failed to parse config", errors.K.IO, io.EOF, "file", "/tmp/app-config.yaml")
	log.Warn("failed to read config, using defaults", err)

	// The same applies for nested eluvio errors...
	log.Warn("call failed",
		errors.E(
			"configuration incomplete: timeout missing",
			errors.K.Invalid,
			err))

	// per-package log configuration
	log.Debug("log is suppressed due to INFO log level")
}
