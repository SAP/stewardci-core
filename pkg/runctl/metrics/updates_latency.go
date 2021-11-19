package metrics

import (
	"sync"
	"time"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// UpdatesLatency is a metric that observes the duration of updates
	// partitioned by a "type".
	UpdatesLatency UpdatesLatencyMetric = &updatesLatency{}
)

// UpdatesLatencyMetric is the interface of UpdatesLatency
type UpdatesLatencyMetric interface {
	Observe(typ string, duration time.Duration)
}

func init() {
	UpdatesLatency.(*updatesLatency).init()
}

type updatesLatency struct {
	initOnlyOnce sync.Once
	metric       *prometheus.HistogramVec
}

func (m *updatesLatency) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				// TODO use metric name prefixes consistently
				// TODO use better name
				//Subsystem: subsystem,
				//Name:      "updates_latency_seconds",
				Name:    "steward_pipelinerun_update_seconds",
				Help:    "A histogram vector of the duration of update operations.",
				Buckets: prometheus.ExponentialBuckets(0.001, 1.3, 30),
			},
			[]string{
				"type",
			},
		)
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *updatesLatency) Observe(typ string, duration time.Duration) {
	labels := prometheus.Labels{
		"type": typ,
	}
	m.metric.With(labels).Observe(duration.Seconds())
}
