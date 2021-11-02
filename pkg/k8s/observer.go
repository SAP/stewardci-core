package k8s

import "time"

// DurationByTypeObserver observes durations by type
type DurationByTypeObserver interface {
	ObserveRetryDurationByType(string, time.Duration)
	ObserveUpdateDurationByType(string, time.Duration)
}
