package runctl

import (
	"strconv"
	"testing"
	"time"

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
	_, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.ErrorContains(t, err, "configmaps \"steward-pipelineruns\" not found")
}

func Test_loadPipelineRunsConfig_NoNetworkConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap( /* no data here */ nil),
	)

	// EXERCISE
	_, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.ErrorContains(t, err, "configmaps \"steward-pipelineruns-network-policies\" not found")
}

func Test_loadPipelineRunsConfig_EmptyConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap( /* no data here */ nil),
		newNetworkPolicyConfigMap(nil),
	)

	// EXERCISE
	resultConfig, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, err)
	expectedConfig := &pipelineRunsConfigStruct{}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_loadPipelineRunsConfig_EmptyEntries(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap(map[string]string{
			pipelineRunsConfigKeyPSCFSGroup:    "",
			pipelineRunsConfigKeyPSCRunAsGroup: "",
			pipelineRunsConfigKeyPSCRunAsUser:  "",
			pipelineRunsConfigKeyLimitRange:    "",
			pipelineRunsConfigKeyResourceQuota: "",
			pipelineRunsConfigKeyTimeout:       "",
		}),
		newNetworkPolicyConfigMap(nil),
	)

	// EXERCISE
	resultConfig, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, err)
	expectedConfig := &pipelineRunsConfigStruct{}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

var metav1Duration = func(d time.Duration) *metav1.Duration {
	return &metav1.Duration{Duration: d}
}

func Test_loadPipelineRunsConfig_CompleteConfigMap(t *testing.T) {
	t.Parallel()

	int64Ptr := func(val int64) *int64 { return &val }

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap(
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
		),
		newNetworkPolicyConfigMap(map[string]string{
			networkPoliciesConfigKeyDefault: "defaultKey",
			"defaultKey":                    "defaultPolicy",
			"foo":                           "fooPolicy",
			"bar":                           "barPolicy",
			"_other_special_key":            "baz",
		}),
	)

	// EXERCISE
	resultConfig, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, err)
	expectedConfig := &pipelineRunsConfigStruct{
		Timeout:              metav1Duration(time.Minute * 4444),
		LimitRange:           "limitRange1",
		ResourceQuota:        "resourceQuota1",
		DefaultNetworkPolicy: "defaultPolicy",
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
			resultConfig, err := loadPipelineRunsConfig(cf)

			// VERIFY
			assert.Assert(t, err != nil)
			assert.Assert(t, resultConfig == nil)
		})
	}
}

// TO BE DISCUSSED how errors occur if only one cm is existing.
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
			AnyTimes()
	}

	// EXERCISE
	resultConfig, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.Assert(t, err == expectedError)
	assert.Assert(t, resultConfig == nil)
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
