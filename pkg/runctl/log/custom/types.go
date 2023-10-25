package custom

// PipelineRunAccessorConfig is the config to defines a way to access a value of a pipeline run
type PipelineRunAccessorConfig struct {
	LogKey string `yaml:"logKey,omitempty"`
	Kind   Kind   `yaml:"kind,omitempty"`
	Spec   Spec   `yaml:"spec,omitempty"`
}

// Kind of accessor
type Kind string

const (
	// KindLabelAccessor defines an accessor for labels
	KindLabelAccessor Kind = "label"

	// KindAnnotationAccessor defines an accessor for annotations
	KindAnnotationAccessor Kind = "annotation"
)

// Spec of the accessor
type Spec struct {
	Key string `yaml:"key,omitempty"`
}
