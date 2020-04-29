package secrets

import (
	"context"
)

type contextKey string

const (
	secretProviderKey contextKey = "secretProvider"
)

// GetSecretProvider returns SecretProvider from the context
// or nil if it doesn't contain one.
func GetSecretProvider(ctx context.Context) SecretProvider {
	result := ctx.Value(secretProviderKey)
	if result == nil {
		return nil
	}
	return result.(SecretProvider)
}

// WithSecretProvider returns Context with SecretProvider
func WithSecretProvider(ctx context.Context, secretProvider SecretProvider) context.Context {
	if secretProvider == nil {
		return ctx
	}
	return context.WithValue(ctx, secretProviderKey, secretProvider)
}
