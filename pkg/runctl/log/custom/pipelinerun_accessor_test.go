package custom

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_parseAccessors_invalidYaml(t *testing.T) {
	t.Parallel()
	const (
		anyKey      = "key1"
		invalidYaml = "invalid1"
	)

	// EXERCISE
	_, err := ParseLoggingDetailsProvider(invalidYaml)

	// VERIFY
	assert.Assert(t, err != nil)
}

func Test_parseAccessors_success(t *testing.T) {
	t.Parallel()
	// SETUP
	const (
		config = "[{logKey: label1, kind: label, spec: {key: key1}},{logKey: label2, kind: annotation, spec: {key: key2}}]"
	)

	run := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"key2": "value2"},
			Labels:      map[string]string{"key1": "value1"},
		},
	}
	// EXERCISE
	result, err := ParseLoggingDetailsProvider(config)

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, 2, len(result))

	logDetails := result[0](run)
	assert.DeepEqual(t, []any{"label1", "value1"}, logDetails)

	logDetails = result[1](run)
	assert.DeepEqual(t, []any{"label2", "value2"}, logDetails)

}
