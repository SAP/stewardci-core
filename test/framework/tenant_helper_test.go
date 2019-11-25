package framework

import (
	"context"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
)

func GetTenantTest(t *testing.T) {
	// SETUP
	ctx := context.Background()
	clientFactory := fake.NewClientFactory()
	ctx = SetClientFactory(ctx, clientFactory)
	ctx = SetNamespace(ctx, "ns1")

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

func DeleteTenantTest(t *testing.T) {
	// SETUP
	ctx := context.Background()
	clientFactory := fake.NewClientFactory()
	ctx = SetClientFactory(ctx, clientFactory)
	ctx = SetNamespace(ctx, "ns1")

	tenantTest := TenantSuccessTest(ctx)
	tenant := tenantTest.tenant
	createdTenant, err := CreateTenant(ctx, tenant)
	assert.NilError(t, err)
	// EXERCISE
	err = DeleteTenant(ctx, createdTenant)
	// VERIFY
	assert.NilError(t, err)
}
