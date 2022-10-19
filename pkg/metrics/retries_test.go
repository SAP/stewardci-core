package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"
)

func Test_retriesMetric(t *testing.T) {
	// no parallel: patching global state

	// SETUP
	reg := prometheus.NewPedanticRegistry()
	t.Cleanup(Testing{}.PatchRegistry(reg))

	examinee := &retriesMetric{}
	examinee.init()

	codeLocation := "codeLocation1"
	retryCount := uint64(88)
	latency := 42 * time.Second

	// EXERCISE
	examinee.Observe(codeLocation, retryCount, latency)

	// VERIFY
	metricFamily, err := reg.Gather()
	assert.NilError(t, err)
	assert.Equal(t, len(metricFamily), 2)

	// latency
	{
		assert.Equal(t, len(metricFamily[0].GetMetric()), 1)

		ioMetric := metricFamily[0].GetMetric()[0]
		// t.Log(ioMetric.Histogram.String())
		assert.Equal(t, ioMetric.Label[0].GetName(), "location")
		assert.Equal(t, ioMetric.Label[0].GetValue(), codeLocation)

		assert.Equal(t, ioMetric.Histogram.GetSampleCount(), uint64(1))
		assert.Equal(t, ioMetric.Histogram.GetSampleSum(), float64(latency.Seconds()))

		for _, bucket := range ioMetric.Histogram.Bucket {
			if float64(latency.Seconds()) <= *bucket.UpperBound {
				assert.Equal(t, *bucket.CumulativeCount, uint64(1))
			} else {
				assert.Equal(t, *bucket.CumulativeCount, uint64(0))
			}
		}
	}

	// retry_count
	{
		assert.Equal(t, len(metricFamily[1].GetMetric()), 1)

		ioMetric := metricFamily[1].GetMetric()[0]
		// t.Log(ioMetric.Histogram.String())
		assert.Equal(t, ioMetric.Label[0].GetName(), "location")
		assert.Equal(t, ioMetric.Label[0].GetValue(), codeLocation)

		assert.Equal(t, ioMetric.Histogram.GetSampleCount(), uint64(1))
		assert.Equal(t, ioMetric.Histogram.GetSampleSum(), float64(retryCount))

		for _, bucket := range ioMetric.Histogram.Bucket {
			if float64(retryCount) <= *bucket.UpperBound {
				assert.Equal(t, *bucket.CumulativeCount, uint64(1))
			} else {
				assert.Equal(t, *bucket.CumulativeCount, uint64(0))
			}
		}
	}
}
