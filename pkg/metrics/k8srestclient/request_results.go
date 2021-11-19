package k8srestclient

import (
	"sync"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	k8sclientmetrics "k8s.io/client-go/tools/metrics"
)

var (
	_ k8sclientmetrics.ResultMetric = (*requestResults)(nil)

	requestResultsInstance *requestResults = &requestResults{}
)

func init() {
	requestResultsInstance.init()
}

// requestResults is the adapter for the `RequestResult` metric of client-go.
type requestResults struct {
	metric       *prometheus.CounterVec
	initOnlyOnce sync.Once
}

func (m *requestResults) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      "request_results",
				Help:      "The number of finished requests partitioned by host, HTTP method and status code.",
			},
			[]string{
				"host",
				"method",
				"status",
			},
		)
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *requestResults) Increment(code string, method string, host string) {
	labels := prometheus.Labels{
		"host":   host,
		"method": method,
		"status": code,
	}
	m.metric.With(labels).Inc()
}
