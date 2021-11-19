package metrics

import (
	"sync"

	stewardapi "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/benbjohnson/clock"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// PipelineRunsPeriodic is a metric that observes all existing pipeline runs
	// periodically.
	PipelineRunsPeriodic PipelineRunsMetric = &pipelineRunsPeriodic{}
)

func init() {
	PipelineRunsPeriodic.(*pipelineRunsPeriodic).init()
}

type pipelineRunsPeriodic struct {
	clock          clock.Clock
	initOnlyOnce   sync.Once
	durationMetric *prometheus.HistogramVec
}

func (m *pipelineRunsPeriodic) init() {
	m.initOnlyOnce.Do(func() {
		if m.clock == nil {
			m.clock = clock.New()
		}

		m.durationMetric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				// TODO use metric name prefixes consistently
				//Subsystem: subsystem,
				//Name:      "ongoing_state_duration_periodic_observations_seconds",
				Name: "steward_pipelinerun_ongoing_state_duration_periodic_observations_seconds",
				Help: "A histogram vector partitioned by pipeline run states that counts the number of periodic observations of pipeline runs in a state grouped by the duration of the state at the time of the observation." +
					"\n\nThe purpose of this metric is the detection of overly long processing times, caused by e.g. hanging controllers." +
					"\n\nThere's one histogram per pipeline run state (label `state`)." +
					" All existing pipeline runs get counted periodically, i.e. every observation cycle counts each pipeline run in exactly one histogram." +
					" This means a single pipeline run is counted zero, one or multiple times in the same or different buckets of the same or different histograms." +
					" This in turn means without knowing the observation and scraping intervals it is not possible to infer the _absolute_ number of pipeline runs observed." +
					" It is only meaningful to calculate a _ratio_ between observations in certain buckets and the total number of observations (in a single or across multiple histograms)." +
					"\n\nPipeline runs that are marked as deleted are not counted to exclude delays caused by finalization.",
				Buckets: prometheus.ExponentialBuckets(60, 2, 7),
			},
			[]string{
				"state",
			},
		)
		metrics.Registerer().MustRegister(m.durationMetric)
	})
}

func (m *pipelineRunsPeriodic) Observe(run *stewardapi.PipelineRun) {
	if m.isNewRun(run) {
		m.observe(stewardapi.StateNew, run.CreationTimestamp)
	} else if run.Status.StartedAt != nil {
		m.observe(run.Status.State, *run.Status.StartedAt)
	}
}

func (m *pipelineRunsPeriodic) observe(state stewardapi.State, since metav1.Time) {
	if since.IsZero() {
		// cannot observe pipeline run if start timestamp is not set
		return
	}
	duration := m.clock.Since(since.Time)
	if duration < 0 {
		// cannot observe pipeline run if start time lies in the future
		return
	}
	labels := prometheus.Labels{
		"state": string(state),
	}
	m.durationMetric.With(labels).Observe(duration.Seconds())
}

func (m *pipelineRunsPeriodic) isNewRun(run *stewardapi.PipelineRun) bool {
	state := run.Status.State
	return false ||
		state == stewardapi.StateUndefined ||
		state == stewardapi.StateNew
}
