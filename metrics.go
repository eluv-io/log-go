package log

import "sync/atomic"

// Metrics is the interface for collecting log metrics (counters for log calls).
type Metrics interface {
	// FileCreated increments the counter for created log files
	FileCreated()
	// InstanceCreated increments the counter for created log objects
	InstanceCreated()
	// Error increments the counter for messages logged with Error level
	Error(logger string)
	// Warn increments the counter for messages logged with Warn level
	Warn(logger string)
	// Info increments the counter for messages logged with Info level
	Info(logger string)
	// Debug increments the counter for messages logged with Debug level
	Debug(logger string)
}

// =============================================================================

var (
	// pMetrics is a pointer to the global metrics instance
	pMetrics  atomic.Pointer[metricsWrapper]
	noMetrics = &noopMetrics{}
)

func init() {
	pMetrics.Store(&metricsWrapper{metrics: noMetrics})
}

// noopMetrics is a no-op Metrics implementation.
type noopMetrics struct{}

func (n *noopMetrics) FileCreated()     {}
func (n *noopMetrics) InstanceCreated() {}
func (n *noopMetrics) Error(string)     {}
func (n *noopMetrics) Warn(string)      {}
func (n *noopMetrics) Info(string)      {}
func (n *noopMetrics) Debug(string)     {}

type metricsWrapper struct {
	metrics Metrics
}

func metrics() Metrics {
	ret := pMetrics.Load()
	if ret == nil {
		// should not happen
		return noMetrics
	}
	return ret.metrics
}

// SetMetrics sets a new global Metrics instance.
func SetMetrics(m Metrics) {
	if m == nil {
		m = noMetrics
	}
	pMetrics.Store(&metricsWrapper{metrics: m})
}
