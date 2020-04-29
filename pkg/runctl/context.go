package runctl

import (
	"context"
)

type contextKey string

const (
	testingKey contextKey = "testing"
)

// getRunInstanceTesting returns runInstanceTesting from the context
// or nil if it doesn't contain one.
func getRunInstanceTesting(ctx context.Context) *runInstanceTesting {
	v := ctx.Value(testingKey)
	if v == nil {
		return nil
	}
	return v.(*runInstanceTesting)
}

// withRunInstanceTesting returns Context with RunInstanceTesting
func withRunInstanceTesting(ctx context.Context, i *runInstanceTesting) context.Context {
	if i == nil {
		return ctx
	}
	return context.WithValue(ctx, testingKey, i)
}
