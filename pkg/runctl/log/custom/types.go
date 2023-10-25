package custom

type pipelineRunAccessorConfig struct {
	LogKey string `yaml:"logKey,omitempty"`
	Kind   kind   `yaml:"kind,omitempty"`
	Spec   Spec   `yaml:"spec,omitempty"`
}

type kind string

const (
	// KindLabelAccessor defines an accessor for labels
	KindLabelAccessor kind = "label"

	// KindAnnotationAccessor defines an accessor for annotations
	KindAnnotationAccessor kind = "annotation"
)

// Spec of the accessor
type Spec struct {
	Key string `yaml:"key,omitempty"`
}
