package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/assert"
)

func Test_DurationObserver(t *testing.T) {
	t.Parallel()

	duration, _ := time.ParseDuration("100ms")

	for _, retry := range []bool{true, false} {
		// SETUP
		m := NewMetrics().(*metrics)
		reg := prometheus.NewPedanticRegistry()
		reg.MustRegister(m.Duration)

		t.Run(fmt.Sprintf("retry_%t", retry), func(t *testing.T) {
			// EXERCISE
			retry := retry
			t.Parallel()

			durationTester(m, duration, retry)

			// VERIFY

			metricFamily, err := reg.Gather()
			assert.NilError(t, err)
			assert.Equal(t, len(metricFamily), 1)
			assert.Equal(t, len(metricFamily[0].GetMetric()), 1)

			ioMetric := metricFamily[0].GetMetric()[0]
			assert.Equal(t, ioMetric.Label[0].GetName(), "caller")
			assert.Equal(t, ioMetric.Label[0].GetValue(), "github.com/SAP/stewardci-core/pkg/metrics.durationTester.func1")
			assert.Equal(t, ioMetric.Label[1].GetName(), "retry")
			assert.Equal(t, ioMetric.Label[1].GetValue(), fmt.Sprintf("%t", retry))

			for _, bucket := range ioMetric.Histogram.Bucket {
				if duration.Seconds() <= *bucket.UpperBound {
					assert.Equal(t, *bucket.CumulativeCount, uint64(1))
				} else {
					assert.Equal(t, *bucket.CumulativeCount, uint64(0))
				}
			}
		})
	}
}

func durationTester(observer DurationObserver, duration time.Duration, retry bool) {
	defer func(start time.Time) {

		elapsed := time.Since(start)
		observer.ObserveDuration(elapsed, retry)

	}(time.Now())

	time.Sleep(duration)
}
