package cfg

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewPipelineRunLabelAccessor(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		key      string
		expected *pipelineRunLabelAccessor
	}{
		{
			name: "empty",
		},
		{
			name: "success",
			key:  "key1",
			expected: &pipelineRunLabelAccessor{
				Key: "key1",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()

			// EXERCISE
			result := NewPipelineRunLabelAccessor(tc.key)

			// VERIFY
			if tc.expected == nil {
				assert.Assert(t, result == nil)
			} else {
				assert.DeepEqual(t, tc.expected, result)
			}
		})
	}
}

func Test_PipelineRunLabelAccessor_access(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		key      string
		labels   map[string]string
		expected string
	}{
		{
			name:     "success",
			key:      "key1",
			labels:   map[string]string{"key1": "value1"},
			expected: "value1",
		},
		{
			name:     "no labels",
			key:      "key1",
			labels:   nil,
			expected: "",
		},
		{
			name:     "empty labels",
			key:      "key1",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "unknown key",
			key:      "key_unknown",
			labels:   map[string]string{"key1": "value1"},
			expected: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()
			// SETUP
			run := &api.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tc.labels,
				},
			}

			examinee := NewPipelineRunLabelAccessor(tc.key)

			// EXERCISE
			result := examinee.Access(run)

			// VERIFY
			assert.Equal(t, tc.expected, result)
		})
	}
}
