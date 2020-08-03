package metrics

import (
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

func fakeStateItem(state api.State, duration time.Duration) *api.StateItem {
	startTime := metav1.Now()
	endTime := metav1.NewTime(startTime.Time.Add(duration))
	return &api.StateItem{
		State:      state,
		StartedAt:  startTime,
		FinishedAt: endTime,
	}
}
