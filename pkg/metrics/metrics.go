package metrics

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
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
	ObserveOngoingStateDuration(state *api.PipelineRun) error
	ObserveDuration(duration time.Duration, retry bool)
	StartServer()
	SetQueueCount(int)
}

type metrics struct {
	Started              prometheus.Counter
	Completed            *prometheus.CounterVec
	StateDuration        *prometheus.HistogramVec
	OngoingStateDuration *prometheus.HistogramVec
	Duration             *prometheus.HistogramVec
	Queued               prometheus.Gauge
	Total                prometheus.Gauge
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
		StateDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipelinerun_state_duration_seconds",
			Help:    "Family of histograms counting the pipeline runs grouped by the duration of their processing states. There's one histogram per pipeline run state (label `state`). A pipeline run gets counted immediately when a state is finished.",
			Buckets: prometheus.ExponentialBuckets(0.125, 2, 15),
		},
			[]string{"state"}),
		OngoingStateDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipelinerun_ongoing_state_duration_periodic_observations_seconds",
			Help:    "Family of histograms counting the number of periodic observations of pipeline runs in certain states grouped by duration of the current state at the time of observation. The purpose of this metric is the detection of overly long processing times, caused by e.g. hanging controllers. There's one histogram per pipeline run state (label `state`). All existing pipeline runs get counted periodically, i.e. every observation cycle counts each pipeline run in exactly one histogram. This means a single pipeline run may be counted zero, one or multiple times in the same or different buckets of a histogram. This in turn means without knowing the observation and scraping intervals it is not possible to infer the _absolute_ number of pipeline runs observed. It is only meaningful to calculate a _ratio_ between observations in certain buckets and the total number of observations (in a single or across all histograms). Pipeline runs that are marked as deleted are not counted to exclude delays caused by finalization.",
			Buckets: prometheus.ExponentialBuckets(60, 2, 7),
		},
			[]string{"state"}),
		Queued: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "steward_queued_total",
			Help: "total queue count of pipelineruns",
		}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "steward_pipelinerun_update_seconds",
			Help:    "pipeline run update duration",
			Buckets: prometheus.ExponentialBuckets(0.01, 1.5, 35),
		},
			[]string{"caller", "retry"}),

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
	prometheus.MustRegister(metrics.StateDuration)
	prometheus.MustRegister(metrics.OngoingStateDuration)
	prometheus.MustRegister(metrics.Duration)
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

// ObserveDurationByState logs duration of the given state
func (metrics *metrics) ObserveDurationByState(state *api.StateItem) error {
	if state.StartedAt.IsZero() {
		return fmt.Errorf("cannot observe StateItem if StartedAt is not set")
	}
	duration := state.FinishedAt.Sub(state.StartedAt.Time)
	if duration < 0 {
		return fmt.Errorf("cannot observe StateItem if FinishedAt is before StartedAt")
	}
	metrics.StateDuration.With(prometheus.Labels{"state": string(state.State)}).Observe(duration.Seconds())
	return nil
}

// ObserveOngoingStateDuration logs the duration of the current (unfinished) pipeline state.
func (metrics *metrics) ObserveOngoingStateDuration(run *api.PipelineRun) error {
	//state undefined is not processed yet and will be metered as new
	if run.Status.State == api.StateUndefined || run.Status.State == api.StateNew {
		if run.CreationTimestamp.IsZero() {
			return fmt.Errorf("cannot observe pipeline run if creationTimestamp is not set")
		}
		duration := time.Now().Sub(run.CreationTimestamp.Time)
		if duration < 0 {
			return fmt.Errorf("cannot observe pipeline run if creationTimestamp is in future")
		}
		metrics.OngoingStateDuration.With(prometheus.Labels{"state": string(api.StateNew)}).Observe(duration.Seconds())
		return nil
	}

	if run.Status.StartedAt.IsZero() {
		return fmt.Errorf("cannot observe StateItem if StartedAt is not set")
	}
	duration := time.Now().Sub(run.Status.StartedAt.Time)
	if duration < 0 {
		return fmt.Errorf("cannot observe StateItem if StartedAt is in the future")
	}
	metrics.OngoingStateDuration.With(prometheus.Labels{"state": string(run.Status.State)}).Observe(duration.Seconds())
	return nil
}

// ObserveDuration logs the duration of updates by type
func (metrics *metrics) ObserveDuration(duration time.Duration, retry bool) {
	caller := callerFunctionName(1, 1)
	metrics.Duration.With(prometheus.Labels{"caller": caller, "retry": fmt.Sprintf("%t", retry)}).Observe(duration.Seconds())
}

// SetQueueCount logs queue count metric
func (metrics *metrics) SetQueueCount(count int) {
	metrics.Queued.Set(float64(count))
}

func callerFunctionName(skip, depth int) string {
	pc := make([]uintptr, 10)
	skipRuntimeCallerAndThisFunction := 2
	entryCount := runtime.Callers(skip+skipRuntimeCallerAndThisFunction, pc)
	if entryCount == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pc[:entryCount])
	var result strings.Builder
	for {
		frame, more := frames.Next()
		result.WriteString(frame.Function)
		if !more {
			break
		}
		depth = depth - 1
		if depth == 0 {
			break
		}
		result.WriteString("->")
	}
	return result.String()
}
