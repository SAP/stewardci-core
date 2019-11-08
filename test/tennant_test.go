// +build e2e

package test

import (
	"testing"

	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

func TestTenantCreation(t *testing.T) {
	t.Parallel()
	clientFactory, namespace := setup(t)
	tenant := builder.Tenant("name", namespace, "displayName")
	stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	tenant, err := stewardClient.Create(tenant)
	assert.NilError(t, err)
        check := CreateTenantCondition(tenant,TenantHasStateResult(api.TenantResultSuccess)) 
        err = WaitForState(clientFactory,check,"tenant_creation")
        assert.NilError(t, err)


}
