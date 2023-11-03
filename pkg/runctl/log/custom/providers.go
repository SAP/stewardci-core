package custom

import "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"

// LoggingDetailsProvider is the type of functions that extract details
// from given pipeline run objects to be added to log entries.
// The returned slice contains key-value pairs. Keys are at even indexes,
// while values are at odd indexes. The slice has an even number of items.
type LoggingDetailsProvider = func(run *v1alpha1.PipelineRun) []any

type loggingDetailConfig struct {
	LogKey string       `yaml:"logKey,omitempty"`
	Kind   providerKind `yaml:"kind,omitempty"`
	Spec   providerSpec `yaml:"spec,omitempty"`
}

type providerKind string

const (
	providerKindLabel      providerKind = "label"
	providerKindAnnotation providerKind = "annotation"
)

// providerSpec is the specification of a provider.
//
// TODO use type-specific specs
type providerSpec struct {
	Key string `yaml:"key,omitempty"`
}

type providerContructorFunc = func(logKey string, spec providerSpec) (LoggingDetailsProvider, error)

var providerRegistry = make(map[providerKind]providerContructorFunc)

// mergeProviders creates a function that calls the given providers in the
// given order and returns the concatenation of their results.
func mergeProviders(providers ...LoggingDetailsProvider) LoggingDetailsProvider {
	return func(run *v1alpha1.PipelineRun) []any {
		result := []any{}
		for _, provider := range providers {
			result = append(result, provider(run)...)
		}
		return result
	}
}
