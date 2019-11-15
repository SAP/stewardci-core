package test

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/test/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantTest is a test for a Tenant
type TenantTest struct {
	name   string
	tenant *api.Tenant
	check  TenantCheck
}

// TenantSuccessTest is a test checking if a tenant was created successfully
func TenantSuccessTest(namespace string) TenantTest {
	return TenantTest{
		name:   "success check",
		tenant: builder.Tenant("name", namespace, "displayName"),
		check:  TenantHasStateResult(api.TenantResultSuccess),
	}
}

// CreateTenant creates a Tenant resource on a client
func CreateTenant(clientFactory k8s.ClientFactory, tenant *api.Tenant) (*api.Tenant, error) {
	stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	return stewardClient.Create(tenant)
}

// GetTenant returns a Tenant resource from a client
func GetTenant(clientFactory k8s.ClientFactory, tenant *api.Tenant) (*api.Tenant, error) {
	stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	return stewardClient.Get(tenant.GetName(), metav1.GetOptions{})
}

// DeleteTenant deletes a Tenant resource from a client
func DeleteTenant(clientFactory k8s.ClientFactory, tenant *api.Tenant) error {
	stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	uid := tenant.GetObjectMeta().GetUID()
	return stewardClient.Delete(tenant.GetName(), &metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &uid},
	})
}
