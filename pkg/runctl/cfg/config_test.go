package cfg

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	serrors "github.com/SAP/stewardci-core/pkg/errors"
	featureflag "github.com/SAP/stewardci-core/pkg/featureflag"
	featureflagtesting "github.com/SAP/stewardci-core/pkg/featureflag/testing"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	corev1clientmocks "github.com/SAP/stewardci-core/pkg/k8s/mocks/client-go/corev1"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const testSystemNamespaceName = "steward-testing"

func init() {
	os.Setenv(system.NamespaceEnvKey, testSystemNamespaceName)
}

func Test_loadPipelineRunsConfig_NoMainConfig(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		/* no main configmap defined here */
		newNetworkPolicyConfigMap(map[string]string{
			networkPoliciesConfigKeyDefault: "key1",
			"key1":                          "policy1",
		}),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

	// VERIFY
	assert.NilError(t, resultErr)
	expectedConfig := &PipelineRunsConfigStruct{
		DefaultNetworkProfile: "key1",
		NetworkPolicies: map[string]string{
			"key1": "policy1",
		},
	}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_loadPipelineRunsConfig_EmptyMainConfig(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		newMainConfigMap( /* no data here */ nil),
		newNetworkPolicyConfigMap(map[string]string{
			networkPoliciesConfigKeyDefault: "key1",
			"key1":                          "policy1",
		}),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

	// VERIFY
	assert.NilError(t, resultErr)
	expectedConfig := &PipelineRunsConfigStruct{
		DefaultNetworkProfile: "key1",
		NetworkPolicies: map[string]string{
			"key1": "policy1",
		},
	}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_loadPipelineRunsConfig_NoNetworkConfig(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		newMainConfigMap(nil),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

	// VERIFY
	assert.Error(t, resultErr, `invalid configuration: ConfigMap "steward-pipelineruns-network-policies" in namespace "steward-testing": is missing`)
	assert.Assert(t, resultConfig == nil)
}

func Test_loadPipelineRunsConfig_EmptyNetworkConfig(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		newNetworkPolicyConfigMap( /* no data here */ nil),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

	// VERIFY
	assert.Error(t, resultErr, `invalid configuration: ConfigMap "steward-pipelineruns-network-policies" in namespace "steward-testing": key "_default" is missing`)
	assert.Assert(t, resultConfig == nil)
}

func Test_loadPipelineRunsConfig_ErrorOnGetMainConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := mocks.NewMockClientFactory(mockCtrl)
	expectedError := errors.New("some error")
	{
		coreV1Ifce := corev1clientmocks.NewMockCoreV1Interface(mockCtrl)
		cf.EXPECT().CoreV1().Return(coreV1Ifce).AnyTimes()
		configMapIfce := corev1clientmocks.NewMockConfigMapInterface(mockCtrl)
		coreV1Ifce.EXPECT().ConfigMaps(gomock.Any()).Return(configMapIfce).AnyTimes()
		configMapIfce.EXPECT().
			Get(ctx, mainConfigMapName, gomock.Any()).
			Return(nil, expectedError).
			Times(1)
	}

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

	// VERIFY
	assert.Assert(t, serrors.IsRecoverable(resultErr))
	assert.Error(t, resultErr, `invalid configuration: ConfigMap "steward-pipelineruns" in namespace "steward-testing": some error`)
	assert.Assert(t, resultConfig == nil)
}

func Test_loadPipelineRunsConfig_ErrorOnGetNetworkPoliciesConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := mocks.NewMockClientFactory(mockCtrl)
	expectedError := errors.New("some error")
	{
		coreV1Ifce := corev1clientmocks.NewMockCoreV1Interface(mockCtrl)
		cf.EXPECT().CoreV1().Return(coreV1Ifce).AnyTimes()
		configMapIfce := corev1clientmocks.NewMockConfigMapInterface(mockCtrl)
		coreV1Ifce.EXPECT().ConfigMaps(gomock.Any()).Return(configMapIfce).AnyTimes()
		configMapIfce.EXPECT().
			Get(ctx, mainConfigMapName, gomock.Any()).
			Return(nil, nil).
			Times(1)
		configMapIfce.EXPECT().
			Get(ctx, networkPoliciesConfigMapName, gomock.Any()).
			Return(nil, expectedError).
			Times(1)
	}

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

	// VERIFY
	assert.Assert(t, serrors.IsRecoverable(resultErr))
	assert.Error(t, resultErr, `invalid configuration: ConfigMap "steward-pipelineruns-network-policies" in namespace "steward-testing": some error`)
	assert.Assert(t, resultConfig == nil)
}

func Test_loadPipelineRunsConfig_CompleteConfig(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		newMainConfigMap(
			map[string]string{
				"_example":                       "exampleString",
				mainConfigKeyLimitRange:          "limitRange1",
				mainConfigKeyResourceQuota:       "resourceQuota1",
				mainConfigKeyPSCRunAsUser:        "1111",
				mainConfigKeyPSCRunAsGroup:       "2222",
				mainConfigKeyPSCFSGroup:          "3333",
				mainConfigKeyTimeout:             "4444m",
				mainConfigKeyTimeoutWait:         "555m",
				mainConfigKeyImage:               "jfrImage1",
				mainConfigKeyImagePullPolicy:     "jfrImagePullPolicy1",
				mainConfigKeyTektonTaskName:      "taskName1",
				mainConfigKeyTektonTaskNamespace: "taskNamespace1",
				"someKeyThatShouldBeIgnored":     "34957349",
			},
		),
		newNetworkPolicyConfigMap(map[string]string{
			networkPoliciesConfigKeyDefault: "networkPolicyKey2",

			"networkPolicyKey1": "networkPolicy1",
			"networkPolicyKey2": "networkPolicy2",
			"networkPolicyKey3": "networkPolicy3",
		}),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

	// VERIFY
	assert.NilError(t, resultErr)
	expectedConfig := &PipelineRunsConfigStruct{
		Timeout:                          metav1Duration(time.Minute * 4444),
		TimeoutWait:                      metav1Duration(time.Minute * 555),
		LimitRange:                       "limitRange1",
		ResourceQuota:                    "resourceQuota1",
		JenkinsfileRunnerImage:           "jfrImage1",
		JenkinsfileRunnerImagePullPolicy: "jfrImagePullPolicy1",
		JenkinsfileRunnerPodSecurityContextRunAsUser:  int64Ptr(1111),
		JenkinsfileRunnerPodSecurityContextRunAsGroup: int64Ptr(2222),
		JenkinsfileRunnerPodSecurityContextFSGroup:    int64Ptr(3333),

		DefaultNetworkProfile: "networkPolicyKey2",
		NetworkPolicies: map[string]string{
			"networkPolicyKey1": "networkPolicy1",
			"networkPolicyKey2": "networkPolicy2",
			"networkPolicyKey3": "networkPolicy3",
		},
		TektonTaskName:      "taskName1",
		TektonTaskNamespace: "taskNamespace1",
	}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_withRecoverablility(t *testing.T) {
	t.Parallel()

	errFoo := fmt.Errorf("foo")

	for _, tc := range []struct {
		name                                  string
		flag, infraError, expectedRecoverable bool
	}{
		{"retry_off_no_infra_error", false, false, false},
		{"retry_off_infra_error", false, true, true},
		{"retry_on_no_infra_error", true, false, true},
		{"retry_on_infra_error", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {

			// SETUP
			defer featureflagtesting.WithFeatureFlag(featureflag.RetryOnInvalidPipelineRunsConfig, tc.flag)()

			// EXERCISE
			resultErr := withRecoverability(errFoo, tc.infraError)

			// VERIFY
			assert.Assert(t, serrors.IsRecoverable(resultErr) == tc.expectedRecoverable)
		})
	}
}

func Test_loadPipelineRunsConfig_InvalidValues(t *testing.T) {
	t.Parallel()

	for i, tc := range []struct {
		key, val string
	}{
		{mainConfigKeyPSCRunAsUser, "a"},
		{mainConfigKeyPSCRunAsUser, "1a"},

		{mainConfigKeyPSCRunAsGroup, "a"},
		{mainConfigKeyPSCRunAsGroup, "1a"},

		{mainConfigKeyPSCFSGroup, "a"},
		{mainConfigKeyPSCFSGroup, "1a"},

		{mainConfigKeyTimeout, "a"},
		{mainConfigKeyTimeout, "1a"},

		{mainConfigKeyTimeoutWait, "a"},
		{mainConfigKeyTimeoutWait, "1a"},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()

			// SETUP
			ctx := context.Background()
			cf := fake.NewClientFactory(
				newMainConfigMap(
					map[string]string{tc.key: tc.val},
				),
				newNetworkPolicyConfigMap(nil),
			)

			// EXERCISE
			resultConfig, resultErr := LoadPipelineRunsConfig(ctx, cf)

			// VERIFY
			assert.Assert(t, resultErr != nil)
			assert.Assert(t, resultConfig == nil)
		})
	}
}

func Test_processMainConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		configData map[string]string
		expected   *PipelineRunsConfigStruct
	}{
		{
			"all_set",
			map[string]string{
				"_example": "exampleString",

				mainConfigKeyTimeout:       "4444m",
				mainConfigKeyTimeoutWait:   "555m",
				mainConfigKeyLimitRange:    "limitRange1",
				mainConfigKeyResourceQuota: "resourceQuota1",

				mainConfigKeyImage:           "jfrImage1",
				mainConfigKeyImagePullPolicy: "jfrImagePullPolicy1",
				mainConfigKeyPSCRunAsUser:    "1111",
				mainConfigKeyPSCRunAsGroup:   "2222",
				mainConfigKeyPSCFSGroup:      "3333",

				"someKeyThatShouldBeIgnored": "34957349",
			},
			&PipelineRunsConfigStruct{
				Timeout:       metav1Duration(time.Minute * 4444),
				TimeoutWait:   metav1Duration(time.Minute * 555),
				LimitRange:    "limitRange1",
				ResourceQuota: "resourceQuota1",

				JenkinsfileRunnerImage:                        "jfrImage1",
				JenkinsfileRunnerImagePullPolicy:              "jfrImagePullPolicy1",
				JenkinsfileRunnerPodSecurityContextRunAsUser:  int64Ptr(1111),
				JenkinsfileRunnerPodSecurityContextRunAsGroup: int64Ptr(2222),
				JenkinsfileRunnerPodSecurityContextFSGroup:    int64Ptr(3333),
			},
		},
		{
			"all_empty",
			map[string]string{
				mainConfigKeyTimeout:       "",
				mainConfigKeyTimeoutWait:   "",
				mainConfigKeyLimitRange:    "",
				mainConfigKeyResourceQuota: "",

				mainConfigKeyImage:           "",
				mainConfigKeyImagePullPolicy: "",
				mainConfigKeyPSCRunAsUser:    "",
				mainConfigKeyPSCRunAsGroup:   "",
				mainConfigKeyPSCFSGroup:      "",
			},
			&PipelineRunsConfigStruct{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()

			// SETUP
			dest := &PipelineRunsConfigStruct{}

			// EXERCISE
			resultErr := processMainConfig(tc.configData, dest)

			// VERIFY
			assert.NilError(t, resultErr)
			assert.DeepEqual(t, tc.expected, dest)
		},
		)
	}
}

func Test_processNetworkPoliciesConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		configData    map[string]string
		expected      *PipelineRunsConfigStruct
		expectedError string
	}{
		{
			"empty",
			map[string]string{},
			&PipelineRunsConfigStruct{},
			`key "_default" is missing`,
		},
		{
			"only_default",
			map[string]string{
				"_default": "key1",
				"key1":     "\r\npolicy1\t",
			},
			&PipelineRunsConfigStruct{
				DefaultNetworkProfile: "key1",
				NetworkPolicies: map[string]string{
					"key1": "\r\npolicy1\t",
				},
			},
			"",
		},
		{
			"default_key_invalid/empty",
			map[string]string{
				"_default": "",
				"":         "policy1",
			},
			&PipelineRunsConfigStruct{},
			`key "_default": value "" is not a valid network policy key`,
		},
		{
			"default_key_invalid/leading_space",
			map[string]string{
				"_default": "\fkey1",
				"key1":     "policy1",
			},
			&PipelineRunsConfigStruct{},
			`key "_default": value "\fkey1" is not a valid network policy key`,
		},
		{
			"default_key_invalid/trailing_space",
			map[string]string{
				"_default": "key1\v",
				"key1":     "policy1",
			},
			&PipelineRunsConfigStruct{},
			`key "_default": value "key1\v" is not a valid network policy key`,
		},
		{
			"default_key_invalid/leading_underscore",
			map[string]string{
				"_default": "_key1",
				"_key1":    "policy1",
			},
			&PipelineRunsConfigStruct{},
			`key "_default": value "_key1" is not a valid network policy key`,
		},
		{
			"default_key_missing/missing",
			map[string]string{
				"_default": "key1",
			},
			&PipelineRunsConfigStruct{},
			`key "_default": value "key1" does not denote an existing network policy key`,
		},
		{
			"default_key_missing/ignored",
			map[string]string{
				"_default": "key1",
				"key1":     " \t\v\r\n\f", // ignored due to blank value
			},
			&PipelineRunsConfigStruct{},
			`key "_default": value "key1" does not denote an existing network policy key`,
		},
		{
			"multiple",
			map[string]string{
				networkPoliciesConfigKeyDefault: "infix whitespace",
				"key1":                          "policy1",
				"key2":                          "policy2",
				"key3":                          "policy3",
				"infix whitespace":              "policy4",
				"leading_whitespace_value":      " \t\v\r\n\fpolicy5",
				"trailing_whitespace_value":     "policy6 \t\v\r\n\f",
			},
			&PipelineRunsConfigStruct{
				DefaultNetworkProfile: "infix whitespace",
				NetworkPolicies: map[string]string{
					"key1":                      "policy1",
					"key2":                      "policy2",
					"key3":                      "policy3",
					"infix whitespace":          "policy4",
					"leading_whitespace_value":  " \t\v\r\n\fpolicy5",
					"trailing_whitespace_value": "policy6 \t\v\r\n\f",
				},
			},
			"",
		},
		{
			"ignored",
			map[string]string{
				networkPoliciesConfigKeyDefault: "defaultKey",
				"defaultKey":                    "a_policy",

				// invalid keys
				"_other_special_key":   "a_policy",
				"":                     "a_policy",
				" leading_whitespace":  "a_policy",
				"trailing_whitespace ": "a_policy",

				// invalid values
				"empty":      "",
				"onlySpaces": " \t\v\r\n\f",
			},
			&PipelineRunsConfigStruct{
				DefaultNetworkProfile: "defaultKey",
				NetworkPolicies: map[string]string{
					"defaultKey": "a_policy",
				},
			},
			"",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()

			// SETUP
			dest := &PipelineRunsConfigStruct{}

			// EXERCISE
			resultErr := processNetworkPoliciesConfig(tc.configData, dest)

			// VERIFY
			if tc.expectedError == "" {
				assert.NilError(t, resultErr)
			} else {
				assert.Equal(t, resultErr.Error(), tc.expectedError)
			}
			assert.DeepEqual(t, tc.expected, dest)
		})
	}
}

func newMainConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mainConfigMapName,
			Namespace: system.Namespace(),
		},
		Data: data,
	}
}

func newNetworkPolicyConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkPoliciesConfigMapName,
			Namespace: system.Namespace(),
		},
		Data: data,
	}
}

func int64Ptr(val int64) *int64 { return &val }
