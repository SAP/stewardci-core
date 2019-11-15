// +build e2e

package test

import (
	"testing"

	"gotest.tools/assert"
)

func Test_PipelineRunSingle(t *testing.T) {
	t.Parallel()
	// SETUP
	clientFactory, namespace, waiter := setup(t)
	test := TenantSuccessTest(namespace)
	tenant := test.tenant
	tenant, err := CreateTenant(clientFactory, tenant)
	assert.NilError(t, err)

	defer DeleteTenant(clientFactory, tenant)
	check := CreateTenantCondition(tenant, test.check, test.name)
	err = waiter.WaitFor(check)
	assert.NilError(t, err)

	tenant, err = GetTenant(clientFactory, tenant)
	assert.NilError(t, err)
	tnn := tenant.Status.TenantNamespaceName

	for _, pipelinerunTestBuilder := range AllTestBuilders {
		pipelineTest := pipelinerunTestBuilder(tnn)
		pr, err := createPipelineRun(clientFactory, pipelineTest.pipelineRun)
		assert.NilError(t, err)
		pipelineRunCheck := CreatePipelineRunCondition(pr, pipelineTest.check, pipelineTest.name)
		err = waiter.WaitFor(pipelineRunCheck)
		assert.NilError(t, err)
	}
}
