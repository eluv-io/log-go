package log

import (
	"time"
)

// Noop is a logger that does nothing.
var Noop ILog = noop{}
var _ = Noop

type noop struct{}

func (n noop) Trace(string, ...interface{})                {}
func (n noop) Debug(string, ...interface{})                {}
func (n noop) Info(string, ...interface{})                 {}
func (n noop) Warn(string, ...interface{})                 {}
func (n noop) Error(string, ...interface{})                {}
func (n noop) Fatal(string, ...interface{})                {}
func (n noop) IsTrace() bool                               { return false }
func (n noop) IsDebug() bool                               { return false }
func (n noop) IsInfo() bool                                { return false }
func (n noop) IsWarn() bool                                { return false }
func (n noop) IsError() bool                               { return false }
func (n noop) IsFatal() bool                               { return false }
func (n noop) Throttle(string, ...time.Duration) Throttled { return n }
