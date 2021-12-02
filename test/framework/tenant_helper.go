package framework

import (
	"context"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	steward "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	"github.com/SAP/stewardci-core/test/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gotest.tools/assert"
)

// TenantTest is a test for a Tenant
type TenantTest struct {
	name   string
	tenant *api.Tenant
	check  TenantCheck
}

// TenantSuccessTest is a test Checking if a tenant was created successfully
func TenantSuccessTest(ctx context.Context) TenantTest {
	return TenantTest{
		name:   "success check",
		tenant: builder.Tenant(GetNamespace(ctx)),
		check:  TenantIsReady(),
	}
}

// CreateTenant creates a Tenant resource on a client
func CreateTenant(ctx context.Context, tenant *api.Tenant) (*api.Tenant, error) {
	return getTenantInterface(ctx).Create(ctx, tenant, metav1.CreateOptions{})
}

// CreateTenantFromJSON creates a Tenant resource on a client
func CreateTenantFromJSON(ctx context.Context, tenantJSON string) (result *api.Tenant, err error) {
	return createTenantFromString(ctx, tenantJSON, "application/json")
}

// CreateTenantFromYAML creates a Tenant resource on a client
func CreateTenantFromYAML(ctx context.Context, tenantYAML string) (result *api.Tenant, err error) {
	return createTenantFromString(ctx, tenantYAML, "application/yaml")
}

func createTenantFromString(ctx context.Context, tenantString string, contentType string) (result *api.Tenant, err error) {
	client := GetClientFactory(ctx).StewardV1alpha1().RESTClient()
	result = &api.Tenant{}
	err = client.Post().
		Namespace(GetNamespace(ctx)).
		Resource("tenants").
		Body([]byte(tenantString)).
		SetHeader("Content-Type", contentType).
		Do(ctx).
		Into(result)
	if err != nil {
		result = nil
	}
	return
}

// GetTenant returns a Tenant resource from a client
func GetTenant(ctx context.Context, tenant *api.Tenant) (*api.Tenant, error) {
	return getTenantInterface(ctx).Get(ctx, tenant.GetName(), metav1.GetOptions{})
}

// DeleteTenant deletes a Tenant resource from a client
func DeleteTenant(ctx context.Context, tenant *api.Tenant) error {
	if tenant == nil {
		return nil
	}
	stewardClient := getTenantInterface(ctx)
	uid := tenant.GetObjectMeta().GetUID()
	return stewardClient.Delete(ctx, tenant.GetName(), metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &uid},
	})
}

func getTenantInterface(ctx context.Context) steward.TenantInterface {
	return GetClientFactory(ctx).StewardV1alpha1().Tenants(GetNamespace(ctx))
}

func ensureTenant(ctx context.Context, t *testing.T) (func(), context.Context) {
	tenantNamespace := GetTenantNamespace(ctx)
	if tenantNamespace == "" {
		tenant := builder.Tenant(GetNamespace(ctx))
		tenant, err := CreateTenant(ctx, tenant)
		assert.NilError(t, err)
		check := CreateTenantCondition(tenant, TenantIsReady())
		_, err = WaitFor(ctx, check)
		assert.NilError(t, err)
		tenant, err = GetTenant(ctx, tenant)
		assert.NilError(t, err)
		return func() { DeleteTenant(ctx, tenant) }, SetTenantNamespace(ctx, tenant.Status.TenantNamespaceName)
	}
	return func() {}, ctx
}
