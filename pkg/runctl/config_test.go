package runctl

import (
	"testing"

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
	resultConfig, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, err)
	expectedConfig := &pipelineRunsConfigStruct{}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_loadPipelineRunsConfig_EmptyConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap( /* no data here */ nil),
	)

	// EXERCISE
	resultConfig, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, err)
	expectedConfig := &pipelineRunsConfigStruct{}
	assert.DeepEqual(t, expectedConfig, resultConfig)
}

func Test_loadPipelineRunsConfig_CompleteConfigMap(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := fake.NewClientFactory(
		newPipelineRunsConfigMap(
			map[string]string{
				"_example":                         "exampleString",
				pipelineRunsConfigKeyNetworkPolicy: "networkPolicy1",
				"someKeyThatShouldBeIgnored":       "34957349",
			},
		),
	)

	// EXERCISE
	resultConfig, err := loadPipelineRunsConfig(cf)

	// VERIFY
	assert.NilError(t, err)
	expectedConfig := &pipelineRunsConfigStruct{
		NetworkPolicy: "networkPolicy1",
	}
	assert.DeepEqual(t, expectedConfig, resultConfig)
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
