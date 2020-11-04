// +build helm

package test

import (
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

func Test_ConfigNetworkPolicies(t *testing.T) {
	t.Parallel()
	template := "templates/config-pipelineruns-network-policies.yaml"

	for _, tc := range []struct {
		name               string
		values             map[string]string
		expectedMapEntries map[string]string
		expectedError      string
	}{
		{"empty",
			map[string]string{},
			map[string]string{"_default": "default"},
			"",
		},
		{"old_policy",
			map[string]string{"pipelineRuns.networkPolicy": "np1"},
			map[string]string{
				"_default": "default",
				"default":  "np1",
			},
			"",
		},
		{"single_policy",
			map[string]string{"pipelineRuns.networkPolicies.key1": "np1"},
			map[string]string{
				"_default": "key1",
				"key1":     "np1",
			},
			"",
		},
		{"single_policy_wrong_default",
			map[string]string{
				"pipelineRuns.networkPolicies.key1":     "np1",
				"pipelineRuns.defaultNetworkPolicyName": "wrongKey1",
			},
			map[string]string{
				"_default": "key1",
				"key1":     "np1",
			},
			"exit status 1",
		},
		{"multi_policy",
			map[string]string{
				"pipelineRuns.defaultNetworkPolicyName": "key2",
				"pipelineRuns.networkPolicies.key1":     "np1",
				"pipelineRuns.networkPolicies.key2":     "np2"},
			map[string]string{
				"_default": "key2",
				"key1":     "np1\n",
				"key2":     "np2"},
			"",
		},
		{"multi_policy_no_default",
			map[string]string{
				"pipelineRuns.networkPolicies.key1": "np1",
				"pipelineRuns.networkPolicies.key2": "np2"},
			map[string]string{},
			"exit status 1",
		},
		{"multi_policy_wrong_default",
			map[string]string{
				"pipelineRuns.defaultNetworkPolicyName": "key3",
				"pipelineRuns.networkPolicies.key1":     "np1",
				"pipelineRuns.networkPolicies.key2":     "np2"},
			map[string]string{},
			"exit status 1",
		},
		{"illegal_default_key",
			map[string]string{
				"pipelineRuns.defaultNetworkPolicyName": "_key3",
				// Without any key old behaviour would be used
				"pipelineRuns.networkPolicies.key1": "np1"},
			map[string]string{},
			"exit status 1",
		},
		{"illegal_key",
			map[string]string{
				"pipelineRuns.networkPolicies._illegal_key": "foo"},
			map[string]string{},
			"exit status 1",
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

				for key, value := range tc.expectedMapEntries {
					assert.Equal(t, cm.Data[key], value, key)
				}
			}
		})
	}
}
