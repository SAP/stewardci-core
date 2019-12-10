package k8s

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardLister "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/cache"
)

func Test__ClientBasedTenantFetcher_ByKey(t *testing.T) {
	factory := fake.NewClientFactory(newTenant(tenant1))
	key := fake.ObjectKey(tenant1, ns1)
	tf, err := NewClientBasedTenantFetcher(factory).ByKey(key)
	assert.Assert(t, tf != nil)
	assert.NilError(t, err)
}
func Test__ClientBasedTenantFetcher_ByKey_NotExisting_ReturnsNilNil(t *testing.T) {
	factory := fake.NewClientFactory()
	tf, err := NewClientBasedTenantFetcher(factory).ByKey("NotExisting1")
	assert.Assert(t, tf == nil)
	assert.NilError(t, err)
}

func Test__ClientBasedTenantFetcher_ByKey_InvalidKey_ReturnsError(t *testing.T) {
	factory := fake.NewClientFactory()
	_, err := NewClientBasedTenantFetcher(factory).ByKey("wrong/key/format")
	assert.Equal(t, `unexpected key format: "wrong/key/format"`, err.Error())
}

func Test__ListerBasedTenantFetcher_ByKey(t *testing.T) {
	lister := createTenantLister(newTenant(tenant1))
	key := fake.ObjectKey(tenant1, ns1)
	tf, err := NewListerBasedTenantFetcher(lister).ByKey(key)
	assert.Assert(t, tf != nil)
	assert.NilError(t, err)
}
func Test__ListerBasedTenantFetcher_ByKey_NotExisting_ReturnsNilNil(t *testing.T) {
	lister := createTenantLister()
	tf, err := NewListerBasedTenantFetcher(lister).ByKey("NotExisting1")
	assert.Assert(t, tf == nil)
	assert.NilError(t, err)
}

func Test__ListerBasedTenantFetcher_ByKey_InvalidKey_ReturnsError(t *testing.T) {
	lister := createTenantLister()
	_, err := NewListerBasedTenantFetcher(lister).ByKey("wrong/key/format")
	assert.Equal(t, `unexpected key format: "wrong/key/format"`, err.Error())
}

func newTenant(name string) *api.Tenant {
	return fake.Tenant(name, name, name, ns1)
}

func createTenantLister(tenants ...*api.Tenant) stewardLister.TenantLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, tenant := range tenants {
		indexer.Add(tenant)
	}
	return stewardLister.NewTenantLister(indexer)
}
