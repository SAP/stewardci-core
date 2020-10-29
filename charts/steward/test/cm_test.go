// +build helm

package test

import (
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_ConfigPipelineruns(t *testing.T) {
	t.Parallel()
	template := "templates/config-pipelineruns.yaml"

	for _, tc := range []struct {
		name               string
		values             map[string]string
		expectedMapEntries map[string]string
		expectedError      string
	}{
		{"full",
			map[string]string{
				"pipelineRuns.jenkinsfileRunner.image":           "repo1:tag1",
				"pipelineRuns.jenkinsfileRunner.imagePullPolicy": "policy1",
			},
			map[string]string{
				"jenkinsfileRunner.image":           "repo1:tag1",
				"jenkinsfileRunner.imagePullPolicy": "policy1",
			},
			"",
		},
		{"imageOnly",
			map[string]string{
				"pipelineRuns.jenkinsfileRunner.image": "repo1:tag1",
			},
			map[string]string{
				"jenkinsfileRunner.image":           "repo1:tag1",
				"jenkinsfileRunner.imagePullPolicy": "IfNotPresent",
			},
			"",
		},

		{"old",
			map[string]string{
				"pipelineRuns.jenkinsfileRunner.image.repository": "repo1",
				"pipelineRuns.jenkinsfileRunner.image.tag":        "tag1",
				"pipelineRuns.jenkinsfileRunner.image.pullPolicy": "policy1",
			},
			map[string]string{},
			"exit status 1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			// EXERCISE
			rendered, err := render(t, template, tc.values)
			if tc.expectedError != "" {
				assert.Assert(t, err != nil)
				log.Printf("Error: %s", err.Error())
				assert.ErrorContains(t, err, tc.expectedError)
			} else {
				assert.NilError(t, err)
				log.Printf("Rendered: %+v", rendered)
				var cm v1.ConfigMap
				helm.UnmarshalK8SYaml(t, rendered, &cm)

				// VERIFY
				for key, value := range tc.expectedMapEntries {
					log.Printf("Key: %s, val: %s, expected: %s", key, cm.Data[key], value)
					assert.Equal(t, cm.Data[key], value, key)
				}
			}
		})
	}
}
