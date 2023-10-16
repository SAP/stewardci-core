package cfg

import (
	yaml "gopkg.in/yaml.v3"
)

// PipelineRunAccessor defines a way to access a value of a pipeline run
type PipelineRunAccessor struct {
	Kind Kind   `yaml:"kind,omitempty"`
	Name string `yaml:"name,omitempty"`
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
	result := map[string]PipelineRunAccessor{}
	var err error
	if strVal, ok := cd[key]; ok && strVal != "" {
		err = yaml.Unmarshal([]byte(strVal), &result)
	}
	return result, err
}
