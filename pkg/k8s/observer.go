package k8s

import "time"

// RetryDurationByTypeObserver observes retries
type RetryDurationByTypeObserver interface {
	ObserveRetryDurationByType(typ string, duration time.Duration)
}
