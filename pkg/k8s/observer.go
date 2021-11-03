package k8s

import "time"

// DurationObserver observes durations by type
type DurationObserver interface {
	ObserveDuration(time.Duration, bool)
}
