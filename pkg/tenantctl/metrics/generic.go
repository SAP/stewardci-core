package metrics

// CounterMetric is a monotonic counter metric.
type CounterMetric interface {
	Inc()
}

// SettableGaugeMetric is a numeric metric that can be set to a value.
type SettableGaugeMetric interface {
	Set(float64)
}
