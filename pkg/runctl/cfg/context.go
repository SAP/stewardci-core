package cfg

import (
	"context"

	"github.com/SAP/stewardci-core/pkg/k8s"
)

type contextKey struct{}

func FromContext(ctx context.Context) (*PipelineRunsConfigStruct, error) {
	if v, ok := ctx.Value(contextKey{}).(*configLoader); ok {
		config, err := v.loadConfig(ctx)
		if err != nil {
			return nil, err
		}
		return config, nil
	}

	return nil, nil
}

func NewContext(ctx context.Context, factory k8s.ClientFactory) context.Context {
	loader := &configLoader{
		factory: factory,
	}
	return context.WithValue(ctx, contextKey{}, loader)
}

func NewContextWithConfig(ctx context.Context, config *PipelineRunsConfigStruct) context.Context {
	loader := &configLoader{
		config: config,
	}
	return context.WithValue(ctx, contextKey{}, loader)
}

type configLoader struct {
	factory k8s.ClientFactory
	config  *PipelineRunsConfigStruct
}

func (c *configLoader) loadConfig(ctx context.Context) (*PipelineRunsConfigStruct, error) {
	if c.config != nil {
		return c.config, nil
	}
	return LoadPipelineRunsConfig(ctx, c.factory)
}
