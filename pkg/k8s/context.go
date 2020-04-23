package k8s

import (
	"context"
)

type contextKey string

const (
	factoryKey                            contextKey = "factory"
	serviceAccountTokenSecretRetrieverKey contextKey = "secretRetriever"
)

// GetClientFactory returns ClientFactory from the context
// or nil if it doesn't contain one.
func GetClientFactory(ctx context.Context) ClientFactory {
	return ctx.Value(factoryKey).(ClientFactory)
}

// WithClientFactory returns Context with ClientFactory
func WithClientFactory(ctx context.Context, factory ClientFactory) context.Context {
	if factory == nil {
		return ctx
	} else {
		return context.WithValue(ctx, factoryKey, factory)
	}
}

// GetServiceAccountTokenSecretRetrieverFromContext provides the
// `ServiceAccountTokenSecretRetriever` instance from the given context,
// or nil if it doesn't contain one.
func GetServiceAccountTokenSecretRetrieverFromContext(ctx context.Context) ServiceAccountTokenSecretRetriever {
	return ctx.Value(serviceAccountTokenSecretRetrieverKey).(ServiceAccountTokenSecretRetriever)
}

func WithServiceAccountTokenSecretRetriever(ctx context.Context, instance ServiceAccountTokenSecretRetriever) context.Context {
	// just demo, real impl must contain nil check,
	// return orig ctx if value is present already, ...
	return context.WithValue(ctx, serviceAccountTokenSecretRetrieverKey, instance)
}
