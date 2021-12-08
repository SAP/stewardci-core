package metrics

import (
	"sync"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ControllerHeartbeats counts the number of heartbeats of
	// the tenant controller.
	ControllerHeartbeats CounterMetric = &controllerHeartbeats{}
)

func init() {
	ControllerHeartbeats.(*controllerHeartbeats).init()
}

type controllerHeartbeats struct {
	initOnlyOnce sync.Once
	metric       prometheus.Counter
}

func (m *controllerHeartbeats) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "controller_heartbeats_total",
				Help:      "The number of heartbeats of the tenant controller instance.",
			},
		)
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *controllerHeartbeats) Inc() {
	m.metric.Inc()
}
