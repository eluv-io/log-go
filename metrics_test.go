package log_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eluv-io/log-go"
)

func TestMetrics(t *testing.T) {
	c := log.NewConfig()
	c.Named = map[string]*log.Config{
		"/dummy": {
			Level:   "debug",
			Handler: "text",
		},
	}
	log.SetDefault(c)

	m := &metrics{}
	log.SetMetrics(m)

	dummy := log.Get("/dummy")

	for i := 0; i < 3; i++ {
		log.Error("message", "f1", "v1")
		log.Warn("message", "f1", "v1")
		log.Info("message", "f1", "v1")
		dummy.Debug("message", "f1", "v1")
	}
	require.Equal(t, 0, m.files)
	require.Equal(t, 1, m.instances)
	require.Equal(t, 3, m.error)
	require.Equal(t, 3, m.warn)
	require.Equal(t, 3, m.info)
	require.Equal(t, 3, m.debug)
}

type metrics struct {
	files, instances, error, warn, info, debug int
}

func (m *metrics) FileCreated()     { m.files++ }
func (m *metrics) InstanceCreated() { m.instances++ }
func (m *metrics) Error(string)     { m.error++ }
func (m *metrics) Warn(string)      { m.warn++ }
func (m *metrics) Info(string)      { m.info++ }
func (m *metrics) Debug(string)     { m.debug++ }
