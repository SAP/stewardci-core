package k8s

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// TenantFetcher has methods to fetch tenants from Kubernetes
type TenantFetcher interface {
	ByKey(key string) (*api.Tenant, error)
}

type tenantFetcher struct {
	factory ClientFactory
}

//NewTenantFetcher retruns an operative implementation of TenantFetcher
func NewTenantFetcher(factory ClientFactory) TenantFetcher {
	return &tenantFetcher{factory: factory}
}

// ByKey fetches Tenant resource from Kubernetes.
//     key    has to be "<namespace>/<name>"
// Return nil,nil if tenant with key does not exist
func (tf *tenantFetcher) ByKey(key string) (*api.Tenant, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	client := tf.factory.StewardV1alpha1().Tenants(namespace)
	t, err := client.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return t, err
}
