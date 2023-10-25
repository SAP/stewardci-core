package custom

import (
	"fmt"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

// NewPipelineRunAnnotationAccessor creates a new PipelineRunAccessor to access
// the annotation with the provided key
func NewPipelineRunAnnotationAccessor(logKey string, spec Spec) (LoggingDetailsProvider, error) {
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
