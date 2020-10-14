package test

import (
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_ConfigNetworkPolicies(t *testing.T) {
	t.Parallel()
	template := "templates/config-network-policies.yaml"

	for _, tc := range []struct {
		name               string
		values             map[string]string
		expectedMapEntries map[string]string
	}{
		{"empty",
			map[string]string{},
			map[string]string{"_default": "default"},
		},
		{"old_policy",
			map[string]string{"pipelineRuns.networkPolicy": "np1"},
			map[string]string{"_default": "default",
				"default": "np1"},
		},
		{"single_policy",
			map[string]string{"pipelineRuns.networkPolicies.key1": "np1"},
			map[string]string{"_default": "key1",
				"key1": "np1"},
		},
		{"multi_policy",
			map[string]string{
				"pipelineRuns.defaultNetworkPolicyName": "key2",
				"pipelineRuns.networkPolicies.key1":     "np1",
				"pipelineRuns.networkPolicies.key2":     "np2"},
			map[string]string{
				"_default": "key2",
				"key1":     "np1",
				"key2":     "np2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			// EXERCISE
			rendered := render(t, template, tc.values)
			log.Printf("Rendered: %+v", rendered)
			var cm v1.ConfigMap
			helm.UnmarshalK8SYaml(t, rendered, &cm)

			// VERIFY
			for key, value := range tc.expectedMapEntries {
				assert.Equal(t, cm.Data[key], value, key)
			}
		})
	}
}
