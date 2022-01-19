package log

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

// SetMetrics sets a new global Metrics instance.
func SetMetrics(m Metrics) {
	if m == nil {
		m = &noopMetrics{}
	}
	metrics = m
}

// =============================================================================

// metrics is the global metrics instance
var metrics Metrics = &noopMetrics{}

// noopMetrics is a no-op Metrics implementation.
type noopMetrics struct{}

func (n *noopMetrics) FileCreated()     {}
func (n *noopMetrics) InstanceCreated() {}
func (n *noopMetrics) Error(string)     {}
func (n *noopMetrics) Warn(string)      {}
func (n *noopMetrics) Info(string)      {}
func (n *noopMetrics) Debug(string)     {}
