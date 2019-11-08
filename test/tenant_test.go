// +build e2e

package test

import (
	"testing"

"github.com/SAP/stewardci-core/pkg/k8s"
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type tenantTest struct {
	name   string
	tenant *api.Tenant
	check  TenantCheck
}

func TenantSuccessTest(namespace string) tenantTest {
	return tenantTest{
		name:   "success check",
		tenant: builder.Tenant("name", namespace, "displayName"),
		check:  TenantHasStateResult(api.TenantResultSuccess),
	}
}

func TestTenantCreation(t *testing.T) {
	t.Parallel()
	clientFactory, namespace, waiter := setup(t)
	test := TenantSuccessTest(namespace)
	tenant := test.tenant
	tenant, err := CreateTenant(clientFactory, tenant)
	assert.NilError(t, err)
        defer DeleteTenant(clientFactory,tenant)	
	check := CreateTenantCondition(tenant, test.check, test.name)
	err = waiter.WaitFor(check)
	assert.NilError(t, err)
}

func CreateTenant (clientFactory k8s.ClientFactory, tenant *api.Tenant) (*api.Tenant,error) {
    stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
        return stewardClient.Create(tenant)
}

func DeleteTenant (clientFactory k8s.ClientFactory, tenant *api.Tenant) (error) {
    stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
        uid := tenant.GetObjectMeta().GetUID()
	return stewardClient.Delete(tenant.GetName(),&metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &uid},
	})
}
