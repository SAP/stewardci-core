package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_ConfigPipelineRunsNetworkPolicies(t *testing.T) {
	t.Parallel()
	template := "templates/config-pipelineruns-network-policies.yaml"

	for _, tc := range []struct {
		name                       string
		values                     map[string]string
		expectedDataEntries        map[string]string
		expectedAdditionalDataKeys []string
		expectedError              string
	}{
		{
			name:   "empty",
			values: map[string]string{},
			expectedDataEntries: map[string]string{
				"_default": "default",
			},
			expectedAdditionalDataKeys: []string{
				"default",
			},
		},
		{
			name: "old_policy",
			values: map[string]string{
				"pipelineRuns.networkPolicy": "np1",
			},
			expectedDataEntries: map[string]string{
				"_default": "default",
				"default":  "np1",
			},
		},
		{
			name: "single_policy",
			values: map[string]string{
				"pipelineRuns.networkPolicies.key1": "np1",
			},
			expectedDataEntries: map[string]string{
				"_default": "key1",
				"key1":     "np1",
			},
		},
		{
			name: "single_policy_wrong_default_key",
			values: map[string]string{
				"pipelineRuns.networkPolicies.key1":     "np1",
				"pipelineRuns.defaultNetworkPolicyName": "wrong_key1",
			},
			expectedError: "exit status 1",
		},
		{
			name: "multi_policy",
			values: map[string]string{
				"pipelineRuns.defaultNetworkPolicyName": "key2",
				"pipelineRuns.networkPolicies.key1":     "np1",
				"pipelineRuns.networkPolicies.key2":     "np2",
			},
			expectedDataEntries: map[string]string{
				"_default": "key2",
				"key1":     "np1\n",
				"key2":     "np2"},
		},
		{
			name: "multi_policy_no_default",
			values: map[string]string{
				"pipelineRuns.networkPolicies.key1": "np1",
				"pipelineRuns.networkPolicies.key2": "np2",
			},
			expectedError: "exit status 1",
		},
		{
			name: "multi_policy_wrong_default",
			values: map[string]string{
				"pipelineRuns.defaultNetworkPolicyName": "key3",
				"pipelineRuns.networkPolicies.key1":     "np1",
				"pipelineRuns.networkPolicies.key2":     "np2",
			},
			expectedError: "exit status 1",
		},
		{
			name: "illegal_default_key",
			values: map[string]string{
				// key must not start with `_`
				"pipelineRuns.defaultNetworkPolicyName": "_key1",
				// At least one network policy must be defined,
				// otherwise the built-in default policy is used.
				"pipelineRuns.networkPolicies.key1": "np1",
			},
			expectedError: "exit status 1",
		},
		{
			name: "illegal_key",
			values: map[string]string{
				// key must not start with `_`
				"pipelineRuns.networkPolicies._key1": "foo",
			},
			expectedError: "exit status 1",
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
				defer func() {
					if t.Failed() {
						t.Logf("Rendered: %+v", rendered)
					}
				}()

				var cm v1.ConfigMap
				helm.UnmarshalK8SYaml(t, rendered, &cm)
				// ignore example entry
				delete(cm.Data, "_example")

				unexpectedDataKeys := make(map[string]struct{}, len(cm.Data))
				for key := range cm.Data {
					unexpectedDataKeys[key] = struct{}{}
				}

				for key, value := range tc.expectedDataEntries {
					delete(unexpectedDataKeys, key)
					assert.Equal(t, cm.Data[key], value, key)
				}
				for _, key := range tc.expectedAdditionalDataKeys {
					delete(unexpectedDataKeys, key)
					_, hasKey := cm.Data[key]
					assert.Assert(t, hasKey, "expected data key does not exist: '%s'", key)
				}

				assert.DeepEqual(t, unexpectedDataKeys, map[string]struct{}{})
			}
		})
	}
}
