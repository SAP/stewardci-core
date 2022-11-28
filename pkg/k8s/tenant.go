package k8s

import (
	"context"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardLister "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// TenantFetcher has methods to fetch tenants from Kubernetes
type TenantFetcher interface {
	// ByKey fetches Tenant resource from Kubernetes.
	//     key    has to be "<namespace>/<name>"
	// Return nil,nil if tenant with key does not exist
	ByKey(ctx context.Context, key string) (*api.Tenant, error)
}

type clientBasedTenantFetcher struct {
	factory ClientFactory
}

// NewClientBasedTenantFetcher returns an operative implementation of TenantFetcher
func NewClientBasedTenantFetcher(factory ClientFactory) TenantFetcher {
	return &clientBasedTenantFetcher{factory: factory}
}

// ByKey implements interface TenantFetcher
func (tf *clientBasedTenantFetcher) ByKey(ctx context.Context, key string) (*api.Tenant, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	client := tf.factory.StewardV1alpha1().Tenants(namespace)
	t, err := client.Get(ctx, name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return t, err
}

type listerBasedTenantFetcher struct {
	lister stewardLister.TenantLister
}

// NewListerBasedTenantFetcher creates a new lister based tenant fetcher
func NewListerBasedTenantFetcher(lister stewardLister.TenantLister) TenantFetcher {
	return &listerBasedTenantFetcher{lister: lister}
}

// ByKey implements interface TenantFetcher
func (l *listerBasedTenantFetcher) ByKey(ctx context.Context, key string) (*api.Tenant, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	lister := l.lister.Tenants(namespace)
	tenant, err := lister.Get(name)
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return tenant, err
}
