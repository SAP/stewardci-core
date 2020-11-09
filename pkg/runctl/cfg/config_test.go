package cfg

import (
	"fmt"
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
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

func Test_loadPipelineRunsConfig_NoConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory( /* no objects */ )

	// EXERCISE
	resultConfig, err := LoadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, err)
	expectedConfig := &PipelineRunsConfigStruct{}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_loadPipelineRunsConfig_NoNetworkConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap( /* no data here */ nil),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, resultErr)
	expectedConfig := &PipelineRunsConfigStruct{}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_loadPipelineRunsConfig_EmptyConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap( /* no data here */ nil),
		newNetworkPolicyConfigMap( /* no data here */ nil),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(cf)

	// VERIFY
	assert.Equal(t, `invalid configuration: ConfigMap "steward-pipelineruns" in namespace "knative-testing": key "_default" is missing or invalid`, resultErr.Error())
	assert.Assert(t, resultConfig == nil)
}

var metav1Duration = func(d time.Duration) *metav1.Duration {
	return &metav1.Duration{Duration: d}
}

func Test_loadPipelineRunsConfig_ErrorOnGetConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
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
			Get(pipelineRunsConfigMapName, gomock.Any()).
			Return(nil, expectedError).
			Times(1)
	}

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(cf)

	// VERIFY
	assert.Assert(t, serrors.IsRecoverable(resultErr))
	assert.Equal(t, resultErr.Error(), expectedError.Error())
	assert.Assert(t, resultConfig == nil)
}

func Test_loadPipelineRunsConfig_ErrorOnGetNetworkPoliciesMap(t *testing.T) {
	t.Parallel()

	// SETUP
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
			Get(pipelineRunsConfigMapName, gomock.Any()).
			Return(nil, nil).
			Times(1)
		configMapIfce.EXPECT().
			Get(networkPoliciesConfigMapName, gomock.Any()).
			Return(nil, expectedError).
			Times(1)
	}

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(cf)

	// VERIFY
	assert.Assert(t, serrors.IsRecoverable(resultErr))
	assert.Equal(t, resultErr.Error(), expectedError.Error())
	assert.Assert(t, resultConfig == nil)
}

func Test_loadPipelineRunsConfig_CompleteConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap(
			map[string]string{
				"_example":                           "exampleString",
				pipelineRunsConfigKeyLimitRange:      "limitRange1",
				pipelineRunsConfigKeyResourceQuota:   "resourceQuota1",
				pipelineRunsConfigKeyPSCRunAsUser:    "1111",
				pipelineRunsConfigKeyPSCRunAsGroup:   "2222",
				pipelineRunsConfigKeyPSCFSGroup:      "3333",
				pipelineRunsConfigKeyTimeout:         "4444m",
				pipelineRunsConfigKeyImage:           "image1",
				pipelineRunsConfigKeyImagePullPolicy: "policy1",
				"someKeyThatShouldBeIgnored":         "34957349",
			},
		),
		newNetworkPolicyConfigMap(map[string]string{
			networkPoliciesConfigKeyDefault: "defaultKey",
			"defaultKey":                    "defaultPolicy",
			"foo":                           "fooPolicy",
			"bar":                           "barPolicy",
			"_other_special_key":            "baz",
			"":                              "emptyKeyWillBeSkipped",
		}),
	)

	// EXERCISE
	resultConfig, resultErr := LoadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, resultErr)
	expectedConfig := &PipelineRunsConfigStruct{
		Timeout:                          metav1Duration(time.Minute * 4444),
		LimitRange:                       "limitRange1",
		ResourceQuota:                    "resourceQuota1",
		JenkinsfileRunnerImage:           "image1",
		JenkinsfileRunnerImagePullPolicy: "policy1",
		DefaultNetworkPolicy:             "defaultPolicy",
		NetworkPolicies: map[string]string{
			"foo": "fooPolicy",
			"bar": "barPolicy",
		},
		JenkinsfileRunnerPodSecurityContextRunAsUser:  int64Ptr(1111),
		JenkinsfileRunnerPodSecurityContextRunAsGroup: int64Ptr(2222),
		JenkinsfileRunnerPodSecurityContextFSGroup:    int64Ptr(3333),
	}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_withRecoverablility(t *testing.T) {
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
			// VALIDATE
			assert.Assert(t, serrors.IsRecoverable(resultErr) == tc.expectedRecoverable)
		})
	}
}

func Test_loadPipelineRunsConfig_InvalidValues(t *testing.T) {
	for i, p := range []struct {
		key, val string
	}{
		{pipelineRunsConfigKeyPSCRunAsUser, "a"},
		{pipelineRunsConfigKeyPSCRunAsUser, "1a"},

		{pipelineRunsConfigKeyPSCRunAsGroup, "a"},
		{pipelineRunsConfigKeyPSCRunAsGroup, "1a"},

		{pipelineRunsConfigKeyPSCFSGroup, "a"},
		{pipelineRunsConfigKeyPSCFSGroup, "1a"},

		{pipelineRunsConfigKeyTimeout, "a"},
		{pipelineRunsConfigKeyTimeout, "1a"},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			p := p // capture current value before going parallel

			t.Parallel()

			// SETUP
			cf := fake.NewClientFactory(
				newPipelineRunsConfigMap(
					map[string]string{p.key: p.val},
				),
				newNetworkPolicyConfigMap(nil),
			)

			// EXERCISE
			resultConfig, resultErr := LoadPipelineRunsConfig(cf)

			// VERIFY
			assert.Assert(t, resultErr != nil)
			assert.Assert(t, resultConfig == nil)
		})
	}
}

func Test_processMainConfig(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configMap map[string]string
		expected  *PipelineRunsConfigStruct
	}{

		{"all_set",
			map[string]string{
				"_example":                         "exampleString",
				pipelineRunsConfigKeyLimitRange:    "limitRange1",
				pipelineRunsConfigKeyResourceQuota: "resourceQuota1",
				pipelineRunsConfigKeyPSCRunAsUser:  "1111",
				pipelineRunsConfigKeyPSCRunAsGroup: "2222",
				pipelineRunsConfigKeyPSCFSGroup:    "3333",
				pipelineRunsConfigKeyTimeout:       "4444m",
				"someKeyThatShouldBeIgnored":       "34957349",
			},
			&PipelineRunsConfigStruct{
				Timeout:       metav1Duration(time.Minute * 4444),
				LimitRange:    "limitRange1",
				ResourceQuota: "resourceQuota1",
				JenkinsfileRunnerPodSecurityContextRunAsUser:  int64Ptr(1111),
				JenkinsfileRunnerPodSecurityContextRunAsGroup: int64Ptr(2222),
				JenkinsfileRunnerPodSecurityContextFSGroup:    int64Ptr(3333),
			},
		},
		{"all_empty",
			map[string]string{
				pipelineRunsConfigKeyPSCFSGroup:    "",
				pipelineRunsConfigKeyPSCRunAsGroup: "",
				pipelineRunsConfigKeyPSCRunAsUser:  "",
				pipelineRunsConfigKeyLimitRange:    "",
				pipelineRunsConfigKeyResourceQuota: "",
				pipelineRunsConfigKeyTimeout:       "",
			},
			&PipelineRunsConfigStruct{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &PipelineRunsConfigStruct{}

			// EXERCISE
			resultErr := processMainConfig(tc.configMap, config)

			// VERIFY
			assert.NilError(t, resultErr)
			assert.DeepEqual(t, tc.expected, config)
		},
		)
	}
}

func Test_processNetworkMap(t *testing.T) {

	for _, tc := range []struct {
		name          string
		networkMap    map[string]string
		expected      *PipelineRunsConfigStruct
		expectedError string
	}{
		{"empty",
			map[string]string{},
			&PipelineRunsConfigStruct{},
			`invalid configuration: ConfigMap "steward-pipelineruns" in namespace "knative-testing": key "_default" is missing or invalid`,
		},
		{"only_default",
			map[string]string{
				"_default":    "default_key",
				"default_key": "default_np",
			},
			&PipelineRunsConfigStruct{
				DefaultNetworkPolicy: "default_np",
			},
			"",
		},

		{"wrong_default_key",
			map[string]string{
				"_default":    "wrong_key1",
				"default_key": "default_np",
			},
			&PipelineRunsConfigStruct{},
			`invalid configuration: ConfigMap "steward-pipelineruns" in namespace "knative-testing": key "_default": no network policy with key "wrong_key1" found`,
		},
		{"multiple_with_correct_default",
			map[string]string{
				networkPoliciesConfigKeyDefault: "defaultKey",
				"defaultKey":                    "defaultPolicy",
				"foo":                           "fooPolicy",
				"bar":                           "barPolicy",
				"_other_special_key":            "baz",
				"":                              "emptyKeyWillBeSkipped",
				" leading_whitespace":           "will be skipped",
				"trailing_whitespace ":          "will be skipped",
				"infix whitespace":              "keep",
			},
			&PipelineRunsConfigStruct{
				DefaultNetworkPolicy: "defaultPolicy",
				NetworkPolicies: map[string]string{
					"foo":              "fooPolicy",
					"bar":              "barPolicy",
					"infix whitespace": "keep",
				},
			},
			"",
		},
		{"skip_empty_values",
			map[string]string{
				networkPoliciesConfigKeyDefault: "defaultKey",
				"defaultKey":                    "defaultPolicy",
				"empty":                         "",
				"onlySpaces": " 	",
			},
			&PipelineRunsConfigStruct{
				DefaultNetworkPolicy: "defaultPolicy",
			},
			"",
		},
		{"default_key_with_whitespace_is_invalid",
			map[string]string{
				networkPoliciesConfigKeyDefault: " defaultKey ",
				"defaultKey":                    "defaultPolicy",
			},
			&PipelineRunsConfigStruct{},
			`invalid configuration: ConfigMap "steward-pipelineruns" in namespace "knative-testing": key "_default" is missing or invalid`,
		},
		{"default_key_with_leading_underscore_is_invalid",
			map[string]string{
				networkPoliciesConfigKeyDefault: "_defaultKey",
			},
			&PipelineRunsConfigStruct{},
			`invalid configuration: ConfigMap "steward-pipelineruns" in namespace "knative-testing": key "_default" is missing or invalid`,
		},
		{"skip_whitespaces_from_empty_default_policy",
			map[string]string{
				networkPoliciesConfigKeyDefault: "defaultKey",
				"defaultKey": " 	",
			},
			&PipelineRunsConfigStruct{},
			`invalid configuration: ConfigMap "steward-pipelineruns" in namespace "knative-testing": key "_default": no network policy with key "defaultKey" found`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// SETUP
			config := &PipelineRunsConfigStruct{}
			// EXERCISE
			resultErr := processNetworkMap(tc.networkMap, config)
			// VERIFY
			if tc.expectedError == "" {
				assert.NilError(t, resultErr)
			} else {
				assert.Equal(t, resultErr.Error(), tc.expectedError)
			}
			assert.DeepEqual(t, tc.expected, config)

		})
	}
}

func newPipelineRunsConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunsConfigMapName,
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
