package run

import (
	"context"
)

type contextKey string

const (
	runManagerKey contextKey = "manager"
)

// GetRunManager returns Manager from context
func GetRunManager(ctx context.Context) Manager {
	result := ctx.Value(runManagerKey)
	if result == nil {
		return nil
	}
	return result.(Manager)
}

// WithRunManager returns Context with Manager
func WithRunManager(ctx context.Context, rm Manager) context.Context {
	return context.WithValue(ctx, runManagerKey, rm)
}
