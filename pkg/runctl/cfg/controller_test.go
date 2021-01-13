package cfg

import (
	"testing"

	serrors "github.com/SAP/stewardci-core/pkg/errors"
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

func Test_loadControllerConfig_config_not_found(t *testing.T) {
	t.Parallel()
	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cf := newErrorFactory(mockCtrl, controllerConfigMapName)
	// EXERCISE
	_, resultErr := LoadControllerConfig(cf)

	// VERIFY
	assert.Assert(t, serrors.IsRecoverable(resultErr))
	assert.Error(t, resultErr, `invalid configuration: ConfigMap "steward-controller" in namespace "knative-testing": some error`)
}

func Test_loadControllerConfig(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		configData     map[string]string
		expectedConfig *ControllerConfigStruct
	}{
		{
			"UpgradeModeEnabled",
			map[string]string{
				"upgradeMode": "true",
			},
			&ControllerConfigStruct{
				UpgradeMode: true,
			},
		},
		{
			"UpgradeModeDisabled",
			map[string]string{
				"upgradeMode": "false",
			},
			&ControllerConfigStruct{
				UpgradeMode: false,
			},
		},
		{
			"UpgradeModeMissing",
			map[string]string{},
			&ControllerConfigStruct{
				UpgradeMode: false,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()
			// SETUP
			cf := fake.NewClientFactory(
				newControllerConfigMap(tc.configData),
			)

			// EXERCISE
			resultConfig, resultErr := LoadControllerConfig(cf)

			// VERIFY
			assert.NilError(t, resultErr)
			assert.DeepEqual(t, tc.expectedConfig, resultConfig)
		})
	}
}

func newControllerConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerConfigMapName,
			Namespace: system.Namespace(),
		},
		Data: data,
	}
}

func newErrorFactory(mockCtrl *gomock.Controller, configMapName string) *mocks.MockClientFactory {
	cf := mocks.NewMockClientFactory(mockCtrl)
	expectedError := errors.New("some error")
	{
		coreV1Ifce := corev1clientmocks.NewMockCoreV1Interface(mockCtrl)
		cf.EXPECT().CoreV1().Return(coreV1Ifce).AnyTimes()
		configMapIfce := corev1clientmocks.NewMockConfigMapInterface(mockCtrl)
		coreV1Ifce.EXPECT().ConfigMaps(gomock.Any()).Return(configMapIfce).AnyTimes()
		configMapIfce.EXPECT().
			Get(configMapName, gomock.Any()).
			Return(nil, expectedError).
			Times(1)
	}
	return cf
}
