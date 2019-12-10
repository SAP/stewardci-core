package framework

import (
	"context"
	"time"
)

const (
	defaultInterval = 1 * time.Second
)

const (
	waitIntervalKey contextKey = "waitInterval"
)

// GetWaitInterval returns the wait Interval
// Defaults to 1s if nothing was set
func GetWaitInterval(ctx context.Context) time.Duration {
	interval := ctx.Value(waitIntervalKey)
	if interval == nil {
		return defaultInterval
	}
	return ctx.Value(waitIntervalKey).(time.Duration)
}

// SetWaitInterval sets the test Name to the context
func SetWaitInterval(ctx context.Context, interval time.Duration) context.Context {
	return context.WithValue(ctx, waitIntervalKey, interval)
}
