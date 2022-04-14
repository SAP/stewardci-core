//go:build frameworktest
// +build frameworktest

package framework

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"fmt"
	"testing"

	"gotest.tools/assert"
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
