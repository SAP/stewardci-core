package metrics

import (
	"sync"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// PipelineRunsStarted counts the pipeline runs that have been started.
	PipelineRunsStarted CounterMetric = &pipelineRunsStarted{}
)

func init() {
	PipelineRunsStarted.(*pipelineRunsStarted).init()
}

type pipelineRunsStarted struct {
	initOnlyOnce sync.Once
	metric       prometheus.Counter
}

func (m *pipelineRunsStarted) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewCounter(prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "started_total",
			Help:      "The total number of started pipeline runs.",
		})
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *pipelineRunsStarted) Inc() {
	m.metric.Inc()
}
