package cfg

import (
	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	yaml "gopkg.in/yaml.v3"
)

// PipelineRunAccessorConfig is the config to defines a way to access a value of a pipeline run
type PipelineRunAccessorConfig struct {
	Kind Kind   `yaml:"kind,omitempty"`
	Name string `yaml:"name,omitempty"`
}

// PipelineRunAccessor defines an interface to access a pipeline run
type PipelineRunAccessor interface {
	Access(run *v1alpha1.PipelineRun) string
}

// Kind of accessor
type Kind string

const (
	// KindLabelAccessor defines an accessor for labels
	KindLabelAccessor Kind = "label"

	// KindAnnotationAccessor defines an accessor for annotations
	KindAnnotationAccessor Kind = "annotation"
)

func (cd configDataMap) parseAccessors(key string) (map[string]PipelineRunAccessor, error) {
	configs := map[string]PipelineRunAccessorConfig{}
	var err error
	if strVal, ok := cd[key]; ok && strVal != "" {
		err = yaml.Unmarshal([]byte(strVal), &configs)
		if err != nil {
			return nil, err
		}
	}
	if len(configs) == 0 {
		return nil, nil
	}
	result := map[string]PipelineRunAccessor{}
	for key, accessorConfig := range configs {
		switch accessorConfig.Kind {
		case KindLabelAccessor:
			accessor := NewPipelineRunLabelAccessor(accessorConfig.Name)
			if accessor != nil {
				result[key] = accessor
			}

		case KindAnnotationAccessor:
			accessor := NewPipelineRunAnnotationAccessor(accessorConfig.Name)
			if accessor != nil {
				result[key] = accessor
			}
		}
	}
	return result, nil
}
