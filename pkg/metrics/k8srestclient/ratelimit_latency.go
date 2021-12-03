package k8srestclient

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	k8sclientmetrics "k8s.io/client-go/tools/metrics"
)

var (
	_ k8sclientmetrics.LatencyMetric = (*rateLimitLatency)(nil)

	rateLimitLatencyInstance *rateLimitLatency = &rateLimitLatency{}
)

func init() {
	rateLimitLatencyInstance.init()
}

// rateLimitLatency is the adapter for the `RequestLatency` metric of client-go.
type rateLimitLatency struct {
	metric       *prometheus.HistogramVec
	initOnlyOnce sync.Once
}

func (m *rateLimitLatency) init() {
	m.initOnlyOnce.Do(func() {

		buckets := func() []float64 {
			list := make([]float64, 0, 18)
			for i := 1.0; i <= 1e+6; i *= 10.0 {
				list = append(list, i, i*5.0)
			}
			return list
		}

		m.metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      "ratelimit_latency_millis",
				Help:      "A histogram vector of client-side late limit latency partitioned by URL scheme, hostname, port, URL path and HTTP method.",
				Buckets:   buckets(),
			},
			[]string{
				"scheme",
				"hostname",
				"port",
				"path",
				"method",
			},
		)
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *rateLimitLatency) Observe(ctx context.Context, method string, u url.URL, latency time.Duration) {
	labels := prometheus.Labels{
		"scheme":   u.Scheme,
		"hostname": u.Hostname(),
		// Set the scheme's default port if none is specified in the URL to avoid
		// having possibly two partitions (with default port, without port) for
		// effectively equal URLs.
		"port":   urlPort(u),
		"path":   u.Path,
		"method": method,
	}
	m.metric.With(labels).Observe(float64(latency.Milliseconds()))
}
