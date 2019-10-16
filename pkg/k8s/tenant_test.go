package k8s

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
)

func Test__ByKey(t *testing.T) {
	factory := fake.NewClientFactory(newTenant(tenant1))
	key := fake.ObjectKey(tenant1, ns1)
	tf, err := NewTenantFetcher(factory).ByKey(key)
	assert.Assert(t, tf != nil)
	assert.NilError(t, err)
}
func Test__ByKey_NotExisting_ReturnsNilNil(t *testing.T) {
	factory := fake.NewClientFactory()
	tf, err := NewTenantFetcher(factory).ByKey("NotExisting1")
	assert.Assert(t, tf == nil)
	assert.NilError(t, err)
}

func Test__ByKey_InvalidKey_ReturnsError(t *testing.T) {
	factory := fake.NewClientFactory()
	_, err := NewTenantFetcher(factory).ByKey("wrong/key/format")
	assert.Equal(t, `unexpected key format: "wrong/key/format"`, err.Error())
}

func newTenant(name string) *api.Tenant {
	return fake.Tenant(name, name, name, ns1)
}
