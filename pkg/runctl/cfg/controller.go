package cfg

import (
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/system"
)

const (
	controllerConfigMapName = "steward-controller"

	controllerConfigKeyUpgradeMode = "upgradeMode"
)

// ControllerConfigStruct is a struct holding the controller configuration.
type ControllerConfigStruct struct {
	// UpgradeMode controles if new pipeline runs are picked up for execution or not
	UpgradeMode bool
}

// LoadControllerConfig loads the controller configuration and returns it.
func LoadControllerConfig(clientFactory k8s.ClientFactory) (*ControllerConfigStruct, error) {
	dest := &ControllerConfigStruct{}

	wrapError := func(cause error) error {
		return errors.Wrapf(cause,
			"invalid configuration: ConfigMap %q in namespace %q",
			controllerConfigMapName,
			system.Namespace(),
		)
	}

	configMapIfce := clientFactory.CoreV1().ConfigMaps(system.Namespace())

	var err error
	configMap, err := configMapIfce.Get(controllerConfigMapName, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, withRecoverability(wrapError(err), true)
	}

	if configMap != nil {
		data := configMap.Data
		dest.UpgradeMode = data[controllerConfigKeyUpgradeMode] == "true"
	}
	return dest, nil
}
