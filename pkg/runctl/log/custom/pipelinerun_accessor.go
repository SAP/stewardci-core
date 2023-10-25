package custom

import (
	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	yaml "gopkg.in/yaml.v3"
)

// LoggingDetailsProvider extracts details from a pipeline run to be added to the log
type LoggingDetailsProvider func(run *v1alpha1.PipelineRun) []any

func ParseLoggingDetailsProvider(strVal string) ([]LoggingDetailsProvider, error) {
	var configs = []pipelineRunAccessorConfig{}
	if strVal != "" {
		err := yaml.Unmarshal([]byte(strVal), &configs)
		if err != nil {
			return nil, err
		}
	}

	if len(configs) == 0 {
		return nil, nil
	}

	accessors := []LoggingDetailsProvider{}
	for _, accessorConfig := range configs {
		switch accessorConfig.Kind {
		case KindLabelAccessor:
			accessor, err := NewPipelineRunLabelAccessor(accessorConfig.LogKey, accessorConfig.Spec)
			if err != nil {
				return nil, err
			}
			if accessor != nil {
				accessors = append(accessors, accessor)
			}

		case KindAnnotationAccessor:
			accessor, err := NewPipelineRunAnnotationAccessor(accessorConfig.LogKey, accessorConfig.Spec)
			if err != nil {
				return nil, err
			}
			if accessor != nil {
				accessors = append(accessors, accessor)
			}
		}
	}
	return accessors, nil
}
