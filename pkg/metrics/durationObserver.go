package metrics

import "time"

// DurationObserver is an Interface providing a observe duration function
type DurationObserver interface {
	ObserveDuration(time.Duration, bool)
}
