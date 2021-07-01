package metrics

import (
	"fmt"
	"net/http"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	klog "k8s.io/klog/v2"
)

//TODO: Move to pipeline run controller

// Metrics provides metrics
type Metrics interface {
	CountStart()
	CountResult(api.Result)
	ObserveTotalDurationByState(state *api.StateItem) error
	ObserveCurrentDurationByState(state *api.PipelineRun) error
	ObserveUpdateDurationByType(kind string, duration time.Duration)
	StartServer()
	SetQueueCount(int)
}

type metrics struct {
	Started         prometheus.Counter
	Completed       *prometheus.CounterVec
	TotalDuration   *prometheus.HistogramVec
	CurrentDuration *prometheus.HistogramVec
	Update          *prometheus.HistogramVec
	Queued          prometheus.Gauge
	Total           prometheus.Gauge
}

// NewMetrics create metrics
func NewMetrics() Metrics {
	return &metrics{
		Started: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "steward_pipelineruns_started_total",
			Help: "total number of started pipelines",
		}),
		Completed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "steward_pipelineruns_completed_total",
			Help: "completed pipelines",
		},
			[]string{"result"}),
		TotalDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipelinerun_finished_status_duration_seconds",
			Help:    "pipeline run durations after they changed their status",
			Buckets: prometheus.ExponentialBuckets(0.125, 2, 15),
		},
			[]string{"state"}),
		CurrentDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipelinerun_ongoing_status_duration_seconds",
			Help:    "pipeline run durations in their current status",
			Buckets: prometheus.ExponentialBuckets(60, 2, 7),
		},
			[]string{"state"}),
		Queued: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "steward_queued_total",
			Help: "total queue count of pipelineruns",
		}),
		Update: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipelinerun_update_seconds",
			Help:    "pipeline run update duration",
			Buckets: prometheus.ExponentialBuckets(0.001, 1.3, 30),
		},
			[]string{"type"}),
		Total: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "steward_pipelineruns_total",
			Help: "total number of pipelineruns",
		}),
	}
}

// StartServer registers metrics and start http listener
func (metrics *metrics) StartServer() {
	prometheus.MustRegister(metrics.Started)
	prometheus.MustRegister(metrics.Completed)
	prometheus.MustRegister(metrics.TotalDuration)
	prometheus.MustRegister(metrics.CurrentDuration)
	prometheus.MustRegister(metrics.Update)
	prometheus.MustRegister(metrics.Queued)
	go provideMetrics()
}

func provideMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		klog.Fatalf("Failed to start metrics server for pipeline run controller:%v", err)
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

// ObserveTotalDurationByState logs duration of the given state
func (metrics *metrics) ObserveTotalDurationByState(state *api.StateItem) error {
	if state.StartedAt.IsZero() {
		return fmt.Errorf("cannot observe StateItem if StartedAt is not set")
	}
	duration := state.FinishedAt.Sub(state.StartedAt.Time)
	if duration < 0 {
		return fmt.Errorf("cannot observe StateItem if FinishedAt is before StartedAt")
	}
	metrics.TotalDuration.With(prometheus.Labels{"state": string(state.State)}).Observe(duration.Seconds())
	return nil
}

// ObserveCurrentDurationByState logs the duration of the current (unfinished) pipeline state.
func (metrics *metrics) ObserveCurrentDurationByState(run *api.PipelineRun) error {
	if run.Status.StartedAt.IsZero() {
		return fmt.Errorf("cannot observe StateItem if StartedAt is not set")
	}
	duration := time.Now().Sub(run.Status.StartedAt.Time)
	if duration < 0 {
		return fmt.Errorf("cannot observe StateItem if StartedAt is in the future")
	}
	metrics.CurrentDuration.With(prometheus.Labels{"state": string(run.Status.State)}).Observe(duration.Seconds())
	return nil
}

func (metrics *metrics) ObserveUpdateDurationByType(typ string, duration time.Duration) {
	metrics.Update.With(prometheus.Labels{"type": typ}).Observe(duration.Seconds())
}

// SetQueueCount logs queue count metric
func (metrics *metrics) SetQueueCount(count int) {
	metrics.Queued.Set(float64(count))
}
