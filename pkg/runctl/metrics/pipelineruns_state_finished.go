package metrics

import (
	"sync"

	stewardapi "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// PipelineRunsStateFinished is a metric that observes the state
	// of pipeline runs that has just been finished.
	PipelineRunsStateFinished StateItemsMetric = &pipelineRunsStateFinished{}
)

func init() {
	PipelineRunsStateFinished.(*pipelineRunsStateFinished).init()
}

type pipelineRunsStateFinished struct {
	initOnlyOnce sync.Once
	metric       *prometheus.HistogramVec
}

func (m *pipelineRunsStateFinished) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				// TODO use metric name prefixes consistently
				//Subsystem: subsystem,
				//Name:      "state_duration_seconds",
				Name: "steward_pipelinerun_state_duration_seconds",
				Help: "A histogram vector partitioned by pipeline run states counting the pipeline runs that finished a state grouped by the state duration." +
					"\n\nThere's one histogram per pipeline run state (label `state`)." +
					" A pipeline run gets counted immediately when a state is finished.",
				Buckets: prometheus.ExponentialBuckets(0.125, 2, 15),
			},
			[]string{
				"state",
			},
		)
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *pipelineRunsStateFinished) Observe(state *stewardapi.StateItem) {
	if state.StartedAt.IsZero() || state.FinishedAt.IsZero() {
		// cannot observe state if timestamps are not set
		return
	}
	duration := state.FinishedAt.Sub(state.StartedAt.Time)
	if duration < 0 {
		return
	}
	m.metric.WithLabelValues(string(state.State)).Observe(duration.Seconds())
}
