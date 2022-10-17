package maintenancemode

import (
	"context"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	corev1clientmocks "github.com/SAP/stewardci-core/pkg/k8s/mocks/client-go/corev1"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

func Test_IsMaintenanceMode_getError_(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cf, configMapIfce := newFactoryWithConfigMapIfce(mockCtrl)
	expectErrorOnGetConfigMap(configMapIfce, api.MaintenanceModeConfigMapName, errors.New("some error"))

	// EXERCISE
	_, resultErr := IsMaintenanceMode(ctx, cf)

	// VERIFY
	assert.Error(t, resultErr, `invalid configuration: ConfigMap "steward-maintenance-mode" in namespace "knative-testing": some error`)
}

func Test_IsMaintenanceMode_get_NotFoundError(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cf, configMapIfce := newFactoryWithConfigMapIfce(mockCtrl)
	expectErrorOnGetConfigMap(configMapIfce, api.MaintenanceModeConfigMapName, k8serrors.NewNotFound(api.Resource("pipelineruns"), ""))

	// EXERCISE
	result, resultErr := IsMaintenanceMode(ctx, cf)

	// VERIFY
	assert.Assert(t, result == false)
	assert.NilError(t, resultErr)
}

func Test_IsMaintenanceMode_configMapHasDeletionTimestamp(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cm := newMaintenanceModeConfigMap(map[string]string{
		"maintenanceMode": "true",
	})
	cm.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cf := fake.NewClientFactory(cm)

	// EXERCISE
	result, resultErr := IsMaintenanceMode(ctx, cf)

	// VERIFY
	assert.Assert(t, result == false)
	assert.NilError(t, resultErr)
}

func Test_loadControllerConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		configData map[string]string
		expected   bool
	}{
		{
			"MaintenanceModeEnabled",
			map[string]string{
				"maintenanceMode": "true",
			},
			true,
		},
		{
			"MaintenanceModeDisabled",
			map[string]string{
				"maintenanceMode": "false",
			},
			false,
		},
		{
			"MaintenanceModeMissing",
			map[string]string{},
			false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc // capture current value before going parallel
			t.Parallel()

			// SETUP
			ctx := context.Background()
			cf := fake.NewClientFactory(
				newMaintenanceModeConfigMap(tc.configData),
			)

			// EXERCISE
			result, resultErr := IsMaintenanceMode(ctx, cf)

			// VERIFY
			assert.NilError(t, resultErr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func newMaintenanceModeConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      api.MaintenanceModeConfigMapName,
			Namespace: system.Namespace(),
		},
		Data: data,
	}
}

func newFactoryWithConfigMapIfce(mockCtrl *gomock.Controller) (*mocks.MockClientFactory, *corev1clientmocks.MockConfigMapInterface) {
	cf := mocks.NewMockClientFactory(mockCtrl)
	coreV1Ifce := corev1clientmocks.NewMockCoreV1Interface(mockCtrl)
	cf.EXPECT().CoreV1().Return(coreV1Ifce).AnyTimes()
	configMapIfce := corev1clientmocks.NewMockConfigMapInterface(mockCtrl)
	coreV1Ifce.EXPECT().ConfigMaps(gomock.Any()).Return(configMapIfce).AnyTimes()

	return cf, configMapIfce
}

func expectErrorOnGetConfigMap(configMapIfce *corev1clientmocks.MockConfigMapInterface, configMapName string, expectedError error) {
	configMapIfce.EXPECT().
		Get(gomock.Any(), configMapName, gomock.Any()).
		Return(nil, expectedError).
		Times(1)
}
