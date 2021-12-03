package metrics

import stewardapi "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"

// CounterMetric is a monotonic counter metric.
type CounterMetric interface {
	Inc()
}

// SettableGaugeMetric is a numeric metric that can be set to a value.
type SettableGaugeMetric interface {
	Set(float64)
}

// PipelineRunsMetric observes pipeline runs.
type PipelineRunsMetric interface {
	Observe(pipelineRun *stewardapi.PipelineRun)
}

// StateItemsMetric observes a StateItem
type StateItemsMetric interface {
	Observe(state *stewardapi.StateItem)
}

// ResultsMetric observes the result of a finished pipeline run.
type ResultsMetric interface {
	Observe(result stewardapi.Result)
}
