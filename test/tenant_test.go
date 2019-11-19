// +build e2e

package test

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func Test_TenantCreation(t *testing.T) {
	t.Parallel()
	ctx := setup(t)
	test := TenantSuccessTest(ctx)
	tenant := test.tenant
	tenant, err := CreateTenant(ctx, tenant)
	assert.NilError(t, err)
	defer DeleteTenant(ctx, tenant)
	ctx = SetTestName(ctx, fmt.Sprintf("Create tenant %s", tenant.GetName()))
	check := CreateTenantCondition(tenant, test.check)
	err = WaitFor(ctx, check)
	assert.NilError(t, err)
}
