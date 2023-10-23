package cfg

import (
	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

type pipelineRunLabelAccessor struct {
	Key string
}

// NewPipelineRunLabelAccessor creates a new PipelineRunAccessor to access
// the label with the provided key
func NewPipelineRunLabelAccessor(key string) PipelineRunAccessor {
	if key == "" {
		return nil
	}
	return &pipelineRunLabelAccessor{
		Key: key,
	}
}

// Access returns the desired label of the pipeline run
func (a *pipelineRunLabelAccessor) Access(run *v1alpha1.PipelineRun) string {
	labels := run.GetLabels()
	if labels == nil || a.Key == "" {
		return ""
	}
	return labels[a.Key]
}
