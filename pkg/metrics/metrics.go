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
	ObserveDurationByState(state *api.StateItem) error
	ObserveUpdateDurationByType(kind string, duration time.Duration)
	StartServer()
	SetQueueCount(int)
	ObservePipelineStateDuration(state *api.PipelineStatus, key string) error
}

type metrics struct {
	Started   prometheus.Counter
	Completed *prometheus.CounterVec
	Duration  *prometheus.HistogramVec
	Update    *prometheus.HistogramVec
	Queued    prometheus.Gauge
	Total     prometheus.Gauge
	State 	  *prometheus.GaugeVec
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
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipelinerun_duration_seconds",
			Help:    "pipeline run durations",
			Buckets: prometheus.ExponentialBuckets(0.125, 2, 15),
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
		State: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "steward_state_duration",
			Help: "duration of states by pipelines",
		},
		[]string{"state","pipeline"}),
	}
}

// StartServer registers metrics and start http listener
func (metrics *metrics) StartServer() {
	prometheus.MustRegister(metrics.Started)
	prometheus.MustRegister(metrics.Completed)
	prometheus.MustRegister(metrics.Duration)
	prometheus.MustRegister(metrics.Update)
	prometheus.MustRegister(metrics.Queued)
	prometheus.MustRegister(metrics.State)
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

func (metrics *metrics) ObserveUpdateDurationByType(typ string, duration time.Duration) {
	metrics.Update.With(prometheus.Labels{"type": typ}).Observe(duration.Seconds())
}

func (metrics *metrics) ObservePipelineStateDuration(state *api.PipelineStatus, key string) error{
	if state.StartedAt.IsZero() {
		return fmt.Errorf("cannot observe StateItem if StartedAt is not set")
	}
	duration := time.Now().Sub(state.StartedAt.Time)
	if duration < 0 {
		return fmt.Errorf("cannot observe StateItem if FinishedAt is before StartedAt")
	}
	metrics.State.With(prometheus.Labels{"state": string(state.State),"pipeline":key}).Add(duration.Seconds())
	return nil
}

// SetQueueCount logs queue count metric
func (metrics *metrics) SetQueueCount(count int) {
	metrics.Queued.Set(float64(count))
}
