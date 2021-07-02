package metrics

import (
	"fmt"
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
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

func Test_ObserveOngoingStateDuration(t *testing.T) {
	m := NewMetrics()
	for _, test := range []struct {
		name                           string
		state                          api.State
		startedAtRelativeToNow         time.Duration
		creationTimestampRelativeToNow time.Duration
		expectedError                  error
	}{
		{
			name:                   "success_with_state_preparing",
			state:                  api.StatePreparing,
			startedAtRelativeToNow: -time.Hour * 1,
		},
		{
			name:                   "failed_when_StartedAt_is_zero",
			state:                  api.StateWaiting,
			startedAtRelativeToNow: 0,
			expectedError:          fmt.Errorf("cannot observe StateItem if StartedAt is not set"),
		},
		{
			name:                   "failed_when_StartedAt_is_in_future",
			state:                  api.StateRunning,
			startedAtRelativeToNow: time.Hour * 1,
			expectedError:          fmt.Errorf("cannot observe StateItem if StartedAt is in the future"),
		},
		{
			// TODO check if it gets metered as api.StateNew
			name:                           "success_when_state_undefined",
			state:                          api.StateUndefined,
			startedAtRelativeToNow:         0,
			creationTimestampRelativeToNow: -time.Hour * 1,
		},
		{
			name:                   "failed_when_state_undefined_has_no_creation_timestamp",
			state:                  api.StateUndefined,
			startedAtRelativeToNow: 0,
			expectedError:          fmt.Errorf("cannot observe pipeline run if creationTimestamp is not set"),
		},
		{
			name:                           "failed_when_state_undefined_creation_timestamp_in_future",
			state:                          api.StateUndefined,
			startedAtRelativeToNow:         0,
			creationTimestampRelativeToNow: time.Hour * 1,
			expectedError:                  fmt.Errorf("cannot observe pipeline run if creationTimestamp is in future"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// SETUP
			run := fakePipelineRun(test.state, test.startedAtRelativeToNow, test.creationTimestampRelativeToNow)
			// EXERCISE
			resultErr := m.ObserveOngoingStateDuration(run)

			// VERIFY
			if test.expectedError == nil {
				assert.NilError(t, resultErr)
			} else {
				assert.Error(t, resultErr, test.expectedError.Error())
			}
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

func fakePipelineRun(state api.State, started time.Duration, creation time.Duration) *api.PipelineRun {
	var startTime metav1.Time
	if started != 0 {
		startTime = metav1.NewTime(metav1.Now().Add(started))
	}

	var meta metav1.ObjectMeta
	if creation != 0 {
		creationTimestamp := metav1.NewTime(metav1.Now().Add(creation))
		meta = metav1.ObjectMeta{CreationTimestamp: creationTimestamp}
	}

	return &api.PipelineRun{
		Status:     api.PipelineStatus{State: state, StartedAt: &startTime},
		ObjectMeta: meta,
	}
}
