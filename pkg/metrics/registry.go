package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	registry = prometheus.NewPedanticRegistry()
)

// Registerer returns the registerer for metrics that should be exported
// by the metrics server
func Registerer() prometheus.Registerer {
	return registry
}
