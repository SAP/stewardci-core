package metrics

import (
	"sync"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// WorkqueueLength reflects the length of the controller workqueue.
	WorkqueueLength SettableGaugeMetric = &workqueueLength{}
)

func init() {
	WorkqueueLength.(*workqueueLength).init()
}

type workqueueLength struct {
	initOnlyOnce sync.Once
	metric       prometheus.Gauge
}

func (m *workqueueLength) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewGauge(prometheus.GaugeOpts{
			// TODO use metric name prefixes consistently
			// TODO use better name
			//Subsystem: subsystem,
			//Name:      "workqueue_length",
			Name: "steward_queued_total",
			Help: "The length of the run controller's workqueue.",
		})
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *workqueueLength) Set(value float64) {
	m.metric.Set(value)
}
