package maintenancemode

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

// IsMaintenanceMode returns true if maintenance mode is set.
func IsMaintenanceMode(clientFactory k8s.ClientFactory) (bool, error) {
	wrapError := func(cause error) error {
		return errors.Wrapf(cause,
			"invalid configuration: ConfigMap %q in namespace %q",
			api.MaintenanceModeConfigMapName,
			system.Namespace(),
		)
	}

	configMapIfce := clientFactory.CoreV1().ConfigMaps(system.Namespace())

	var err error
	configMap, err := configMapIfce.Get(api.MaintenanceModeConfigMapName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return true, wrapError(err)
	}

	if configMap != nil {
		data := configMap.Data
		return data[api.MaintenanceModeKeyName] == "true", nil
	}
	return false, nil
}
