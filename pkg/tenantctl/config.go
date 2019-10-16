package tenantctl

import (
	"errors"
	"strconv"

	stewarderrors "github.com/SAP/stewardci-core/pkg/errors"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	keyTenantNamespacePrefix   = "tenant-namespace-prefix"
	keyTenantRole              = "tenant-role"
	keyTenantRandomLengthBytes = "tenant-random-length-bytes"
)

type configData struct {
	randomLengthBytes     int
	tenantNamespacePrefix string
	tenantRoleName        k8s.RoleName
}

type config interface {
	GetRandomLengthBytesOrDefault(defaultValue int) int
	GetTenantNamespacePrefix() string
	GetTenantRoleName() k8s.RoleName
}

//getConfig Returns the steward-client-specific configurartion
func getConfig(factory k8s.ClientFactory, stewardClientNamespace string) (config, error) {
	if stewardClientNamespace == "" {
		return nil, errors.New("GetConfig failed - client namespace not specified")
	}
	newConfig := configData{
		randomLengthBytes: -1,
	}
	//Client config
	if factory != nil {
		namespace, err := factory.CoreV1().Namespaces().Get(stewardClientNamespace, metav1.GetOptions{})
		if err != nil {
			return nil, stewarderrors.Errorf(err, "GetConfig failed - could not get namespace")
		}
		annotations := namespace.GetAnnotations()

		newConfig.tenantNamespacePrefix = annotations[keyTenantNamespacePrefix]
		if newConfig.tenantNamespacePrefix == "" {
			return nil, errors.New(keyTenantNamespacePrefix + " not configured for client")
		}

		newConfig.tenantRoleName = k8s.RoleName(annotations[keyTenantRole])
		if string(newConfig.tenantRoleName) == "" {
			return nil, errors.New(keyTenantRole + " not configured for client")
		}

		if value := annotations[keyTenantRandomLengthBytes]; value != "" {
			i, err := strconv.Atoi(value)
			if err == nil {
				newConfig.randomLengthBytes = i
			}
		}
	}
	return &newConfig, nil
}

func (c *configData) GetRandomLengthBytesOrDefault(defaultValue int) int {
	if c.randomLengthBytes >= 0 {
		return c.randomLengthBytes
	}
	return defaultValue
}

func (c *configData) GetTenantNamespacePrefix() string {
	return c.tenantNamespacePrefix
}

func (c *configData) GetTenantRoleName() k8s.RoleName {
	return c.tenantRoleName
}
