package runctl

import (
	"context"
)

type contextKey string

const (
	testingKey contextKey = "testing"
)

// GetRunInstanceTesting returns runInstanceTesting from the context
// or nil if it doesn't contain one.
func GetRunInstanceTesting(ctx context.Context) *runInstanceTesting {
	return ctx.Value(testingKey).(*runInstanceTesting)
}

// WithRunInstanceTesting returns Context with RunInstanceTesting
func WithRunInstanceTesting(ctx context.Context, i *runInstanceTesting) context.Context {
	if i == nil {
		return ctx
	}
	return context.WithValue(ctx, testingKey, i)
}
