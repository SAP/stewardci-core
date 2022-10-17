package framework

import (
	"context"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/v3/assert"
)

func setupClientContext() context.Context {
	ctx := context.Background()
	clientFactory := fake.NewClientFactory()
	ctx = SetClientFactory(ctx, clientFactory)
	return SetNamespace(ctx, "ns1")
}

func Test_GetTenant(t *testing.T) {
	// SETUP
	ctx := setupClientContext()

	tenantTest := TenantSuccessTest(ctx)
	tenant := tenantTest.tenant
	createdTenant, err := CreateTenant(ctx, tenant)
	assert.NilError(t, err)
	// EXERCISE
	lodedTenant, err := GetTenant(ctx, createdTenant)
	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, "ns1", lodedTenant.GetNamespace())
}

func Test_DeleteTenant(t *testing.T) {
	// SETUP
	ctx := setupClientContext()

	tenantTest := TenantSuccessTest(ctx)
	tenant := tenantTest.tenant
	createdTenant, err := CreateTenant(ctx, tenant)
	assert.NilError(t, err)
	// EXERCISE
	err = DeleteTenant(ctx, createdTenant)
	// VERIFY
	assert.NilError(t, err)
}
