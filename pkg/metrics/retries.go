package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Retries observes finished retried operations.
	Retries RetriesMetric = &retriesMetric{}
)

func init() {
	Retries.(*retriesMetric).init()
}

// RetriesMetric observes finished retried operations.
type RetriesMetric interface {
	// Observe performs a single observation of a finished retry loop.
	// codeLocation is a string representation of the code location
	// of the retry loop.
	// retryCount is the number of retries performed, where the very
	// first attempt is not counted as retry.
	// latency is the elapsed time from the start of the retry loop
	// until is has been finished.
	Observe(codeLocation string, retryCount uint64, latency time.Duration)
}

type retriesMetric struct {
	initOnlyOnce  sync.Once
	countMetric   *prometheus.HistogramVec
	latencyMetric *prometheus.HistogramVec
}

func (m *retriesMetric) init() {
	m.initOnlyOnce.Do(func() {

		countBuckets := func() []float64 {
			return append(
				prometheus.LinearBuckets(1, 1, 9),
				prometheus.ExponentialBuckets(10, 2, 11)...,
			)
		}

		m.countMetric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: Subsystem,
				Name:      "retries_retrycount",
				Help:      "Generic metric for retry loops collecting the number of retries performed for retried operations.",
				Buckets:   countBuckets(),
			},
			[]string{
				"location",
			},
		)
		Registerer().MustRegister(m.countMetric)

		latencyBuckets := func() []float64 {
			list := make([]float64, 0, 18)
			for i := 1e-3; i <= 1e+5; i *= 10.0 {
				list = append(list, i, i*5.0)
			}
			return list
		}

		m.latencyMetric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: Subsystem,
				Name:      "retries_latency_seconds",
				Help:      "Generic metric for retry loops collecting the latency (in seconds) caused by retrying operations.",
				Buckets:   latencyBuckets(),
			},
			[]string{
				"location",
			},
		)
		Registerer().MustRegister(m.latencyMetric)
	})
}

func (m *retriesMetric) Observe(codeLocation string, retryCount uint64, latency time.Duration) {
	if retryCount > 0 {
		m.countMetric.WithLabelValues(codeLocation).Observe(float64(retryCount))
		m.latencyMetric.WithLabelValues(codeLocation).Observe(float64(latency.Seconds()))
	}
}
