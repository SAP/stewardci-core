package test

import (
	"context"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
)

type contextKey string

const (
	factoryKey     contextKey = "factory"
	pipelineRunKey contextKey = "pipelineRun"
	namespaceKey   contextKey = "namespace"
)

// GetClientFactory returns the client factory from the context
func GetClientFactory(ctx context.Context) k8s.ClientFactory {
	return ctx.Value(factoryKey).(k8s.ClientFactory)
}

// SetClientFactory returns a context with client factory
func SetClientFactory(ctx context.Context, clientFactory k8s.ClientFactory) context.Context {
	return context.WithValue(ctx, factoryKey, clientFactory)
}

// GetNamespace returns the namespace from the context
func GetNamespace(ctx context.Context) string {
	return ctx.Value(namespaceKey).(string)
}

// SetNamespace sets the namespace to the context
func SetNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, namespaceKey, namespace)
}

// GetPipelineRun returns the pipeline run from the context
func GetPipelineRun(ctx context.Context) *api.PipelineRun {
	return ctx.Value(pipelineRunKey).(*api.PipelineRun)
}

// SetPipelineRun sets the namespace to the context
func SetPipelineRun(ctx context.Context, pipelineRun *api.PipelineRun) context.Context {
	return context.WithValue(ctx, pipelineRunKey, pipelineRun)
}
