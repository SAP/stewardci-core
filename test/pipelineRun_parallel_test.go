// +build loadtest

package test

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func Test_PipelineRuns(t *testing.T) {
	t.Parallel()
	executePipelineRunTests(t,
		testPlan{testBuilder: PipelineRunSleep,
			parallel: 1,
		},
		testPlan{testBuilder: PipelineRunFail,
			parallel: 2,
		},
		testPlan{testBuilder: PipelineRunOK,
			parallel: 3,
		},
	)
}

func executePipelineRunTests(t *testing.T, testPlans ...testPlan) {
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
	for _, testPlan := range testPlans {
		testPlan := testPlan
		for i := 1; i <= testPlan.parallel; i++ {
			testBuilder := testPlan.testBuilder
			pipelineTest := testBuilder(tnn)
			pipelineTest.name =
				fmt.Sprintf("%s_%d", pipelineTest.name, i)
				//    t.Run(pipelineTest.name,func(t *testing.T) {
				//      pipelineTest := pipelineTest
				//      t.Parallel()
			pr, err := createPipelineRun(clientFactory, pipelineTest.pipelineRun)
			assert.NilError(t, err)
			pipelineRunCheck := CreatePipelineRunCondition(pr, pipelineTest.check, pipelineTest.name)
			err = waiter.WaitFor(pipelineRunCheck)
			assert.NilError(t, err)
			//	})
		}
	}
}
