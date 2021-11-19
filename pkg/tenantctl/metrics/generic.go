package metrics

// SettableGaugeMetric is a numeric metric that can be set to a value.
type SettableGaugeMetric interface {
	Set(float64)
}
