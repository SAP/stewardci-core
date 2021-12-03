package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_ConfigPipelineRuns(t *testing.T) {
	t.Parallel()
	template := "templates/config-pipelineruns.yaml"

	for _, tc := range []struct {
		name               string
		values             map[string]string
		expectedMapEntries map[string]string
		expectedError      string
	}{
		{
			name: "full",
			values: map[string]string{
				"pipelineRuns.jenkinsfileRunner.image":           "repo1:tag1",
				"pipelineRuns.jenkinsfileRunner.imagePullPolicy": "policy1",
			},
			expectedMapEntries: map[string]string{
				"jenkinsfileRunner.image":           "repo1:tag1",
				"jenkinsfileRunner.imagePullPolicy": "policy1",
			},
			expectedError: "",
		},
		{
			name: "imageOnly",
			values: map[string]string{
				"pipelineRuns.jenkinsfileRunner.image": "repo1:tag1",
			},
			expectedMapEntries: map[string]string{
				"jenkinsfileRunner.image":           "repo1:tag1",
				"jenkinsfileRunner.imagePullPolicy": "IfNotPresent",
			},
			expectedError: "",
		},
		{
			name: "old",
			values: map[string]string{
				"pipelineRuns.jenkinsfileRunner.image.repository": "repo1",
				"pipelineRuns.jenkinsfileRunner.image.tag":        "tag1",
				"pipelineRuns.jenkinsfileRunner.image.pullPolicy": "policy1",
			},
			expectedMapEntries: map[string]string{},
			expectedError:      "exit status 1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			// EXERCISE
			rendered, err := render(t, template, tc.values)

			// VERIFY
			if tc.expectedError != "" {
				assert.Assert(t, err != nil)
				t.Logf("Error: %s", err.Error())
				assert.ErrorContains(t, err, tc.expectedError)
			} else {
				assert.NilError(t, err)
				t.Logf("Rendered: %+v", rendered)
				var cm v1.ConfigMap
				helm.UnmarshalK8SYaml(t, rendered, &cm)

				// VERIFY
				for key, value := range tc.expectedMapEntries {
					assert.Equal(t, cm.Data[key], value, key)
				}
			}
		})
	}
}
