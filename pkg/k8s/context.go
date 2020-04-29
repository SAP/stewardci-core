package k8s

import (
	"context"
)

type contextKey string

const (
	factoryKey                            contextKey = "factory"
	serviceAccountTokenSecretRetrieverKey contextKey = "secretRetriever"
	namespaceManagerKey                   contextKey = "namespaceManager"
)

// GetNamespaceManager returns NamespaceManager from the context
// or nil if it doesn't contain one.
func GetNamespaceManager(ctx context.Context) NamespaceManager {
	result := ctx.Value(namespaceManagerKey)
	if result == nil {
		return nil
	}
	return result.(NamespaceManager)
}

// WithNamespaceManager returns Context with NamespaceManager
func WithNamespaceManager(ctx context.Context, namespaceManager NamespaceManager) context.Context {
	if namespaceManager == nil {
		return ctx
	}
	return context.WithValue(ctx, namespaceManagerKey, namespaceManager)
}

// GetClientFactory returns ClientFactory from the context
// or nil if it doesn't contain one.
func GetClientFactory(ctx context.Context) ClientFactory {
	result := ctx.Value(factoryKey)
	if result == nil {
		return nil
	}
	return result.(ClientFactory)
}

// WithClientFactory returns Context with ClientFactory
func WithClientFactory(ctx context.Context, factory ClientFactory) context.Context {
	if factory == nil {
		return ctx
	}
	return context.WithValue(ctx, factoryKey, factory)
}

// GetServiceAccountTokenSecretRetriever provides the
// `ServiceAccountTokenSecretRetriever` instance from the given context,
// or nil if it doesn't contain one.
func GetServiceAccountTokenSecretRetriever(ctx context.Context) ServiceAccountTokenSecretRetriever {
	result := ctx.Value(serviceAccountTokenSecretRetrieverKey)
	if result == nil {
		return nil
	}
	return result.(ServiceAccountTokenSecretRetriever)
}

// WithServiceAccountTokenSecretRetriever returns Context with ServiceAccountTokenSecretRetriever
func WithServiceAccountTokenSecretRetriever(ctx context.Context, instance ServiceAccountTokenSecretRetriever) context.Context {
	// just demo, real impl must contain nil check,
	// return orig ctx if value is present already, ...
	return context.WithValue(ctx, serviceAccountTokenSecretRetrieverKey, instance)
}
