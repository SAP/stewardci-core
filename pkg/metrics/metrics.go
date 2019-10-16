package metrics

import (
	"fmt"
	"log"
	"net/http"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//TODO: Move to pipeline run controller

// Metrics provides metrics
type Metrics interface {
	CountStart()
	CountResult(api.Result)
	ObserveDurationByState(state *api.StateItem) error
	StartServer()
}

type metrics struct {
	Started   prometheus.Counter
	Completed *prometheus.CounterVec
	Duration  *prometheus.HistogramVec
}

// NewMetrics create metrics
func NewMetrics() Metrics {
	return &metrics{
		Started: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "steward_pipeline_runs_started_total_count",
			Help: "total number of started pipelines",
		}),
		Completed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "steward_pipeline_runs_completed_total_count",
			Help: "completed pipelines",
		},
			[]string{"result"}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipeline_run_duration_seconds",
			Help:    "pipeline run durations",
			Buckets: prometheus.ExponentialBuckets(0.125, 2, 15),
		},
			[]string{"state"}),
	}
}

// StartServer registers metrics and start http listener
func (metrics *metrics) StartServer() {
	prometheus.MustRegister(metrics.Started)
	prometheus.MustRegister(metrics.Completed)
	prometheus.MustRegister(metrics.Duration)
	go provideMetrics()
}

func provideMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatalf("Failed to start metrics server for pipeline run controller:%v", err)
	}
}

// CountStart counts the start events
func (metrics *metrics) CountStart() {
	metrics.Started.Inc()
}

// CountResult counts the completed events by result type
func (metrics *metrics) CountResult(result api.Result) {
	metrics.Completed.With(prometheus.Labels{"result": string(result)}).Inc()
}

// ObserveDurationByState logs duration of the state
func (metrics *metrics) ObserveDurationByState(state *api.StateItem) error {
	if state.StartedAt.IsZero() {
		return fmt.Errorf("cannot observe StateItem if StartedAt is not set")
	}
	duration := state.FinishedAt.Sub(state.StartedAt.Time)
	if duration < 0 {
		return fmt.Errorf("cannot observe StateItem if FinishedAt is before StartedAt")
	}
	metrics.Duration.With(prometheus.Labels{"state": string(state.State)}).Observe(duration.Seconds())
	return nil
}
