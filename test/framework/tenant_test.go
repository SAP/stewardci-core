//go:build frameworktest
// +build frameworktest

package framework

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func Test_TenantCreation(t *testing.T) {
	t.Parallel()
	ctx := Setup(t)
	test := TenantSuccessTest(ctx)
	tenant := test.tenant
	tenant, err := CreateTenant(ctx, tenant)
	assert.NilError(t, err)
	defer DeleteTenant(ctx, tenant)
	ctx = SetTestName(ctx, fmt.Sprintf("Create tenant %s", tenant.GetName()))
	check := CreateTenantCondition(tenant, test.check)
	_, err = WaitFor(ctx, check)
	assert.NilError(t, err)
}
