package metrics

import (
	"sync"

	stewardapi "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// PipelineRunsResult counts the number of pipeline runs by result type.
	PipelineRunsResult ResultsMetric = &pipelineRunsResult{}
)

func init() {
	PipelineRunsResult.(*pipelineRunsResult).init()
}

type pipelineRunsResult struct {
	initOnlyOnce sync.Once
	metric       *prometheus.CounterVec
}

func (m *pipelineRunsResult) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "completed_total",
				Help:      "The number of completed pipeline runs partitioned by result type.",
			},
			[]string{
				"result",
			},
		)
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *pipelineRunsResult) Observe(result stewardapi.Result) {
	m.metric.WithLabelValues(string(result)).Inc()
}
