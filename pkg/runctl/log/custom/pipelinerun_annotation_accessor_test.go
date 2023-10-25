package custom

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewPipelineRunAnnotationAccessor(t *testing.T) {
	t.Parallel()
	const logKey = "logKey1"
	for _, tc := range []struct {
		name        string
		spec        Spec
		annotations map[string]string
		expected    string
	}{
		{
			name:        "success",
			spec:        Spec{Key: "key1"},
			annotations: map[string]string{"key1": "value1"},
			expected:    "value1",
		},
		{
			name:        "no annotations",
			spec:        Spec{Key: "key1"},
			annotations: nil,
			expected:    "",
		},
		{
			name:        "empty annotations",
			spec:        Spec{Key: "key1"},
			annotations: map[string]string{},
			expected:    "",
		},
		{
			name:        "unknown key",
			spec:        Spec{Key: "key_unknown"},
			annotations: map[string]string{"key1": "value1"},
			expected:    "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()
			// SETUP
			run := &api.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tc.annotations,
				},
			}

			examinee, err := NewPipelineRunAnnotationAccessor(logKey, tc.spec)
			assert.NilError(t, err)

			// EXERCISE
			result := examinee(run)

			// VERIFY
			assert.Equal(t, logKey, result[0])
			assert.Equal(t, tc.expected, result[1])
		})
	}
}
