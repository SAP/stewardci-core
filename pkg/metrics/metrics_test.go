package metrics

import (
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Duration_Missing_Start_Time(t *testing.T) {
	m := NewMetrics()
	e := m.ObserveDurationByState(&api.StateItem{})
	assert.Equal(t, "cannot observe StateItem if StartedAt is not set", e.Error())
}

func Test_Duration_Missing_End_Time(t *testing.T) {
	m := NewMetrics()
	e := m.ObserveDurationByState(&api.StateItem{StartedAt: metav1.Now()})
	assert.Equal(t, "cannot observe StateItem if FinishedAt is before StartedAt", e.Error())
}

func Test_Duration_End_Before_Beginning(t *testing.T) {
	m := NewMetrics()
	e := m.ObserveDurationByState(fakeStateItem(api.StateRunning, -1))
	assert.Equal(t, "cannot observe StateItem if FinishedAt is before StartedAt", e.Error())
}

func Test_ObserveUpdateDurationByType(t *testing.T) {
	m := NewMetrics()
	m.ObserveUpdateDurationByType("foo", 1)
}

func Test_ObserveOngoingStateDuration_Success(t *testing.T) {
	for _, test := range []struct {
		name          string
		state         api.State
		stateDuration time.Duration
		setStartedAt  bool
		expectedState api.State
	}{
		{
			name:          "success_with_state_preparing",
			state:         api.StatePreparing,
			stateDuration: time.Hour * 2,
			setStartedAt:  true,
			expectedState: api.StatePreparing,
		},
		{
			name:          "success_with_state_cleaning",
			state:         api.StateCleaning,
			stateDuration: time.Minute * 25,
			setStartedAt:  true,
			expectedState: api.StateCleaning,
		},
		{
			name:          "success_when_state_undefined",
			state:         api.StateUndefined,
			stateDuration: time.Hour * 2,
			setStartedAt:  false,
			expectedState: api.StateNew,
		},
		{
			name:          "success_when_state_new",
			state:         api.StateNew,
			stateDuration: time.Hour * 2,
			setStartedAt:  false,
			expectedState: api.StateNew,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// SETUP
			m := NewMetrics().(*metrics)
			reg := prometheus.NewPedanticRegistry()
			reg.MustRegister(m.OngoingStateDuration)

			run := fakePipelineRun(test.state, test.stateDuration, test.setStartedAt)

			// EXERCISE
			resultErr := m.ObserveOngoingStateDuration(run)

			// VERIFY
			assert.NilError(t, resultErr)

			metricFamily, err := reg.Gather()
			assert.NilError(t, err)
			assert.Equal(t, len(metricFamily), 1)
			assert.Equal(t, len(metricFamily[0].GetMetric()), 1)

			ioMetric := metricFamily[0].GetMetric()[0]
			assert.Equal(t, ioMetric.Label[0].GetName(), "state")
			assert.Equal(t, ioMetric.Label[0].GetValue(), string(test.expectedState))

			for _, bucket := range ioMetric.Histogram.Bucket {
				duration := test.stateDuration.Seconds()
				if duration <= *bucket.UpperBound {
					assert.Equal(t, *bucket.CumulativeCount, uint64(1))
				} else {
					assert.Equal(t, *bucket.CumulativeCount, uint64(0))
				}
			}
		})
	}
}

func Test_ObserveOngoingStateDuration_Failures(t *testing.T) {
	for _, test := range []struct {
		name          string
		state         api.State
		stateDuration time.Duration
		setStartedAt  bool
		expectedError string
	}{
		{
			name:          "failed_when_StartedAt_is_zero",
			state:         api.StateWaiting,
			stateDuration: 0,
			setStartedAt:  true,
			expectedError: "cannot observe StateItem if StartedAt is not set",
		},
		{
			name:          "failed_when_StartedAt_is_in_future",
			state:         api.StateRunning,
			stateDuration: -time.Hour * 2,
			setStartedAt:  true,
			expectedError: "cannot observe StateItem if StartedAt is in the future",
		},
		{
			name:          "failed_when_state_undefined_has_no_creation_timestamp",
			state:         api.StateUndefined,
			stateDuration: 0,
			setStartedAt:  false,
			expectedError: "cannot observe pipeline run if creationTimestamp is not set",
		},
		{
			name:          "failed_when_state_undefined_creation_timestamp_in_future",
			state:         api.StateUndefined,
			stateDuration: -time.Hour * 2,
			setStartedAt:  false,
			expectedError: "cannot observe pipeline run if creationTimestamp is in future",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// SETUP
			m := NewMetrics().(*metrics)
			reg := prometheus.NewPedanticRegistry()
			reg.MustRegister(m.OngoingStateDuration)

			run := fakePipelineRun(test.state, test.stateDuration, test.setStartedAt)

			// EXERCISE
			resultErr := m.ObserveOngoingStateDuration(run)

			// VERIFY
			assert.Error(t, resultErr, test.expectedError)

			metricFamily, err := reg.Gather()
			assert.NilError(t, err)
			assert.Equal(t, len(metricFamily), 0)
		})
	}
}

func fakeStateItem(state api.State, duration time.Duration) *api.StateItem {
	startTime := metav1.Now()
	endTime := metav1.NewTime(startTime.Time.Add(duration))
	return &api.StateItem{
		State:      state,
		StartedAt:  startTime,
		FinishedAt: endTime,
	}
}

func fakePipelineRun(state api.State, duration time.Duration, setStartedAt bool) *api.PipelineRun {
	var startTime metav1.Time
	var meta metav1.ObjectMeta

	if duration != 0 {
		if setStartedAt {
			startTime = metav1.NewTime(metav1.Now().Add(-duration))
		} else {
			creationTimestamp := metav1.NewTime(metav1.Now().Add(-duration))
			meta = metav1.ObjectMeta{CreationTimestamp: creationTimestamp}
		}
	}

	return &api.PipelineRun{
		Status:     api.PipelineStatus{State: state, StartedAt: &startTime},
		ObjectMeta: meta,
	}
}
