package framework

import (
	"context"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/google/uuid"
)

type contextKey string

const (
	factoryKey     contextKey = "factory"
	pipelineRunKey contextKey = "PipelineRun"
	namespaceKey   contextKey = "Namespace"
	testNameKey    contextKey = "testName"
	realmUUIDKey   contextKey = "realmUUID"
)

// GetClientFactory returns the client factory from the context
func GetClientFactory(ctx context.Context) k8s.ClientFactory {
	return ctx.Value(factoryKey).(k8s.ClientFactory)
}

// SetClientFactory returns a context with client factory
func SetClientFactory(ctx context.Context, clientFactory k8s.ClientFactory) context.Context {
	return context.WithValue(ctx, factoryKey, clientFactory)
}

// GetNamespace returns the client namespace from the context
func GetNamespace(ctx context.Context) string {
	return ctx.Value(namespaceKey).(string)
}

// SetNamespace sets the client namespace to the context
func SetNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, namespaceKey, namespace)
}

// GetPipelineRun returns the pipeline run from the context
func GetPipelineRun(ctx context.Context) *api.PipelineRun {
	return ctx.Value(pipelineRunKey).(*api.PipelineRun)
}

// SetPipelineRun sets the Namespace to the context
func SetPipelineRun(ctx context.Context, PipelineRun *api.PipelineRun) context.Context {
	return context.WithValue(ctx, pipelineRunKey, PipelineRun)
}

// GetTestName returns the test Name from the context
func GetTestName(ctx context.Context) string {
	return ctx.Value(testNameKey).(string)
}

// SetTestName sets the test Name to the context
func SetTestName(ctx context.Context, Name string) context.Context {
	return context.WithValue(ctx, testNameKey, Name)
}

// GetRealmUUID returns an UUID for the test execution
func GetRealmUUID(ctx context.Context) string {
	val := ctx.Value(realmUUIDKey)
	if val == nil {
		return ""
	}
	return val.(string)
}

// SetRealmUUID returns an UUID for the test execution
func SetRealmUUID(ctx context.Context) context.Context {
	return context.WithValue(ctx, realmUUIDKey, uuid.New().String())
}
