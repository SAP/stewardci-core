package test

import (
	"context"
        "log"
        "time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
        "k8s.io/apimachinery/pkg/util/wait"
)

const (
        interval = 1 * time.Second
        timeout  = 2 * time.Minute
)

type contextKey string

const (
	factoryKey     contextKey = "factory"
	pipelineRunKey contextKey = "pipelineRun"
	namespaceKey   contextKey = "namespace"
        testNameKey    contextKey = "testName"
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

// GetTestName returns the test name from the context
func GetTestName(ctx context.Context) string {
        return ctx.Value(testNameKey).(string)
}

// SetTestName sets the test name to the context
func SetTestName(ctx context.Context, name string) context.Context {
        return context.WithValue(ctx, testNameKey, name)
}


// WaitFor waits for a condition
// it returns an error if condition is not fullfilled
func WaitFor(ctx context.Context, condition WaitCondition) error {
        startTime := time.Now()
        log.Printf("wait for %s", GetTestName(ctx))
        err := wait.PollImmediate(interval, timeout, func() (bool, error) {
                return condition.Check(ctx)
        })
        log.Printf("waiting completed for %s after %s", GetTestName(ctx), time.Now().Sub(startTime))
        return err
}

