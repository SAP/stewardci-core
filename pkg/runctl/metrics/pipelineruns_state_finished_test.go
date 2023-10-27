package metrics

import (
	"strconv"
	"testing"
	"time"

	stewardapi "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"
	k8sapiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_PipelineRunsStateFinished_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *(PipelineRunsStateFinished.(*pipelineRunsStateFinished)) != pipelineRunsStateFinished{})
}

func Test_pipelineRunsStateFinished_Valid(t *testing.T) {
	// no parallel: patching global state

	// SETUP
	reg := prometheus.NewPedanticRegistry()
	t.Cleanup(metrics.Testing{}.PatchRegistry(reg))

	examinee := &pipelineRunsStateFinished{}
	examinee.init()

	stateName := dummyStateName1
	startTime := time.Unix(1000, 0)
	duration := 12_345_678 * time.Microsecond
	endTime := startTime.Add(duration)

	stateItem := &stewardapi.StateItem{
		State:      stewardapi.State(stateName),
		StartedAt:  k8sapiv1.Time{Time: startTime},
		FinishedAt: k8sapiv1.Time{Time: endTime},
	}

	// EXERCISE
	examinee.Observe(stateItem)

	// VERIFY
	metricFamily, err := reg.Gather()
	assert.NilError(t, err)
	assert.Equal(t, len(metricFamily), 2)

	// current metric
	{
		assert.Equal(t, len(metricFamily[0].GetMetric()), 1)

		ioMetric := metricFamily[0].GetMetric()[0]
		//t.Log(ioMetric.Histogram.String())

		stateLabel := ioMetric.Label[0]
		assert.Equal(t, stateLabel.GetName(), "state")
		assert.Equal(t, stateLabel.GetValue(), stateName)

		assert.Equal(t, *ioMetric.Histogram.SampleCount, uint64(1))

		for _, bucket := range ioMetric.Histogram.Bucket {
			if duration.Seconds() <= *bucket.UpperBound {
				assert.Equal(t, *bucket.CumulativeCount, uint64(1))
			} else {
				assert.Equal(t, *bucket.CumulativeCount, uint64(0))
			}
		}
	}

	// deprecated metric
	{
		assert.Equal(t, len(metricFamily[1].GetMetric()), 1)

		ioMetric := metricFamily[1].GetMetric()[0]
		//t.Log(ioMetric.Histogram.String())

		stateLabel := ioMetric.Label[0]
		assert.Equal(t, stateLabel.GetName(), "state")
		assert.Equal(t, stateLabel.GetValue(), stateName)

		assert.Equal(t, *ioMetric.Histogram.SampleCount, uint64(1))

		for _, bucket := range ioMetric.Histogram.Bucket {
			if duration.Seconds() <= *bucket.UpperBound {
				assert.Equal(t, *bucket.CumulativeCount, uint64(1))
			} else {
				assert.Equal(t, *bucket.CumulativeCount, uint64(0))
			}
		}
	}
}

func Test_pipelineRunsStateFinished_Invalid(t *testing.T) {
	// no parallel: patching global state

	for i, tc := range []struct {
		startTime time.Time
		endTime   time.Time
	}{
		{
			// no start time
			// no end time
		},
		{
			startTime: time.Unix(1000, 0),
			// no end time
		},
		{
			// no start time
			endTime: time.Unix(1000, 0),
		},
		{
			startTime: time.Unix(1000, 0),
			// end time before start time
			endTime: time.Unix(999, 0),
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			// no parallel: patching global state

			// SETUP
			reg := prometheus.NewPedanticRegistry()
			t.Cleanup(metrics.Testing{}.PatchRegistry(reg))

			examinee := &pipelineRunsStateFinished{}
			examinee.init()

			stateItem := &stewardapi.StateItem{
				State:      stewardapi.State(dummyStateName1),
				StartedAt:  k8sapiv1.Time{Time: tc.startTime},
				FinishedAt: k8sapiv1.Time{Time: tc.endTime},
			}

			// EXERCISE
			examinee.Observe(stateItem)

			// VERIFY
			metricFamily, err := reg.Gather()
			assert.NilError(t, err)
			assert.Equal(t, len(metricFamily), 0) // no data
		})
	}
}
