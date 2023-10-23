package cfg

import (
	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

type pipelineRunAnnotationAccessor struct {
	Key string
}

// NewPipelineRunAnnotationAccessor creates a new PipelineRunAccessor to access
// the annotation with the provided key
func NewPipelineRunAnnotationAccessor(key string) PipelineRunAccessor {
	if key == "" {
		return nil
	}
	return &pipelineRunAnnotationAccessor{
		Key: key,
	}
}

// Access returns the desired annotation of the pipeline run
func (a *pipelineRunAnnotationAccessor) Access(run *v1alpha1.PipelineRun) string {
	annotations := run.GetAnnotations()
	if annotations == nil || a.Key == "" {
		return ""
	}
	return annotations[a.Key]
}
