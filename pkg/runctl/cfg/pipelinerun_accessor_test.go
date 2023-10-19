package cfg

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_parseAccessors_invalidYaml(t *testing.T) {
	t.Parallel()
	const (
		anyKey      = "key1"
		invalidYaml = "invalid1"
	)
	// SETUP
	cd := configDataMap{anyKey: invalidYaml}

	// EXERCISE
	_, err := cd.parseAccessors(anyKey)

	// VERIFY
	assert.Assert(t, err != nil)
}

func Test_parseAccessors_success(t *testing.T) {
	t.Parallel()
	const (
		anyKey = "key1"
		config = "label1: {kind: label, name: key1}\nlabel2: {kind: annotation, name: key2}"
	)
	// SETUP
	cd := configDataMap{anyKey: config}

	// EXERCISE
	result, err := cd.parseAccessors(anyKey)

	// VERIFY
	expected := map[string]PipelineRunAccessor{
		"label1": &pipelineRunLabelAccessor{Key: "key1"},
		"label2": &pipelineRunAnnotationAccessor{Key: "key2"},
	}
	assert.NilError(t, err)
	assert.DeepEqual(t, expected, result)
}
