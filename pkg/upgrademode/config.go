package upgrademode

import (
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const (
	upgradeModeConfigMapName = "steward-upgrade-mode"

	upgradeModeKeyName = "upgradeMode"
)

// IsUpgradeMode returns true if upgrade mode is set.
func IsUpgradeMode(clientFactory k8s.ClientFactory) (bool, error) {
	wrapError := func(cause error) error {
		return errors.Wrapf(cause,
			"invalid configuration: ConfigMap %q in namespace %q",
			upgradeModeConfigMapName,
			system.Namespace(),
		)
	}

	configMapIfce := clientFactory.CoreV1().ConfigMaps(system.Namespace())

	var err error
	configMap, err := configMapIfce.Get(upgradeModeConfigMapName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return true, wrapError(err)
	}

	if configMap != nil {
		data := configMap.Data
		return data[upgradeModeKeyName] == "true", nil
	}
	return false, nil
}
