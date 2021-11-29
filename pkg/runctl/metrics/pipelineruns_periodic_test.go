package metrics

import (
	"strconv"
	"testing"
	"time"

	stewardapi "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/benbjohnson/clock"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_PipelineRunsPeriodic_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *(PipelineRunsPeriodic.(*pipelineRunsPeriodic)) != pipelineRunsPeriodic{})
}

func Test_PipelineRunsPeriodic_Observe(t *testing.T) {
	// SETUP
	run := &stewardapi.PipelineRun{}
	run.ObjectMeta.SetCreationTimestamp(metav1.NewTime(fakeNow))

	// EXERCISE
	PipelineRunsPeriodic.Observe(run)
}

func Test_pipelineRunsPeriodic_NewRun(t *testing.T) {
	// no parallel: patching global state

	idx := -1

	for _, tc := range []struct {
		omitCreationTime  bool
		state             stewardapi.State
		duration          time.Duration // must be a small duration
		expectObservation bool
	}{
		{
			omitCreationTime:  false,
			duration:          1 * time.Second,
			expectObservation: true,
		},
		{
			omitCreationTime:  false,
			duration:          0 * time.Second,
			expectObservation: true,
		},
		{
			omitCreationTime:  true,
			duration:          1 * time.Second,
			expectObservation: false,
		},
		{
			omitCreationTime:  true,
			duration:          -1 * time.Second,
			expectObservation: false,
		},
	} {
		for _, state := range []stewardapi.State{
			stewardapi.StateUndefined,
			stewardapi.StateNew,
		} {
			tc := tc
			tc.state = state
			idx++

			t.Run(strconv.Itoa(idx), func(t *testing.T) {
				// no parallel: patching global state

				defer func() {
					if t.Failed() {
						t.Logf("tc was: %+v", tc)
					}
				}()

				// SETUP
				mockClock := clock.NewMock()
				mockClock.Set(fakeNow)

				run := &stewardapi.PipelineRun{}
				run.Status.State = tc.state
				if !tc.omitCreationTime {
					run.CreationTimestamp = metav1.NewTime(mockClock.Now().Add(-tc.duration))
				}
				{
					// Start time should be ignored. Set it to see whether it's used anyway.
					t := metav1.NewTime(mockClock.Now().Add(24 * time.Hour))
					run.Status.StartedAt = &t
				}

				// EXERCISE and VERIFY
				doTestPipelineRunsPeriodic(
					t,
					mockClock,
					run,
					tc.duration,
					tc.expectObservation,
					stewardapi.StateNew,
				)
			})
		}
	}
}

func Test_pipelineRunsPeriodic_NonNewRun(t *testing.T) {
	// no parallel: patching global state

	idx := -1

	for _, tc := range []struct {
		omitStartTime     bool
		duration          time.Duration // must be a small duration
		expectObservation bool
	}{
		{
			omitStartTime:     false,
			duration:          1 * time.Second,
			expectObservation: true,
		},
		{
			omitStartTime:     false,
			duration:          0 * time.Second,
			expectObservation: true,
		},
		{
			omitStartTime:     true,
			duration:          1 * time.Second,
			expectObservation: false,
		},
		{
			omitStartTime:     true,
			duration:          -1 * time.Second,
			expectObservation: false,
		},
	} {
		for _, state := range []stewardapi.State{
			stewardapi.StatePreparing,
			stewardapi.StateWaiting,
			stewardapi.StateRunning,
			stewardapi.StateCleaning,
			stewardapi.StateFinished,
			stewardapi.State("testdummy5489674598"),
		} {
			idx++
			t.Run(strconv.Itoa(idx), func(t *testing.T) {
				// no parallel: patching global state

				defer func() {
					if t.Failed() {
						t.Logf("tc was: %+v", tc)
						t.Logf("state was: %#v", state)
					}
				}()

				// SETUP
				mockClock := clock.NewMock()
				mockClock.Set(fakeNow)

				run := &stewardapi.PipelineRun{}
				run.Status.State = state
				// Creation time should be ignored. Set it to see whether it's used anyway.
				run.CreationTimestamp = metav1.NewTime(mockClock.Now().Add(-24 * time.Hour))
				if !tc.omitStartTime {
					t := metav1.NewTime(mockClock.Now().Add(-tc.duration))
					run.Status.StartedAt = &t
				}

				// EXERCISE and VERIFY
				doTestPipelineRunsPeriodic(
					t,
					mockClock,
					run,
					tc.duration,
					tc.expectObservation,
					state,
				)
			})
		}
	}
}

func doTestPipelineRunsPeriodic(
	t *testing.T,
	mockClock *clock.Mock,
	run *stewardapi.PipelineRun,
	duration time.Duration,
	expectObservation bool,
	expectedStateLabelValue stewardapi.State,
) {
	t.Helper()

	// SETUP
	reg := prometheus.NewPedanticRegistry()
	t.Cleanup(metrics.Testing{}.PatchRegistry(reg))

	examinee := &pipelineRunsPeriodic{
		clock: mockClock,
	}
	examinee.init()

	// EXERCISE
	examinee.Observe(run)

	// VERIFY
	metricFamily, err := reg.Gather()
	assert.NilError(t, err)

	if expectObservation {
		assert.Equal(t, len(metricFamily), 2)

		// current metric
		{
			assert.Equal(t, len(metricFamily[0].GetMetric()), 1)

			ioMetric := metricFamily[0].GetMetric()[0]
			//t.Log(ioMetric.Histogram.String())

			stateLabel := ioMetric.Label[0]
			assert.Equal(t, stateLabel.GetName(), "state")
			assert.Equal(t, stateLabel.GetValue(), string(expectedStateLabelValue))

			for _, bucket := range ioMetric.Histogram.Bucket {
				durationSecs := duration.Seconds()
				if durationSecs <= *bucket.UpperBound {
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
			assert.DeepEqual(t, ioMetric, metricFamily[0].GetMetric()[0])
		}
	} else {
		assert.Equal(t, len(metricFamily), 0)
	}
}
