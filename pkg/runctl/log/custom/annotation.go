package custom

import (
	"fmt"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

func init() {
	providerRegistry[providerKindAnnotation] = newAnnotationProvider
}

func newAnnotationProvider(logKey string, spec providerSpec) (LoggingDetailsProvider, error) {
	if logKey == "" || spec.Key == "" {
		return nil, fmt.Errorf("logKey and spec.key must not be nil")
	}
	return func(run *v1alpha1.PipelineRun) []any {
		labels := run.GetAnnotations()
		result := ""
		if labels != nil {
			result = labels[spec.Key]
		}
		return []any{logKey, result}
	}, nil
}
