package test

import (
        "fmt"
        "log"
        "testing"

"gotest.tools/assert"
)

func executePipelineRunTests(t *testing.T, testPlans ...testPlan) {
        clientFactory, namespace, waiter := setup(t)
        test := TenantSuccessTest(namespace)
        tenant := test.tenant
        tenant, err := CreateTenant(clientFactory, tenant)
        assert.NilError(t, err)

        defer DeleteTenant(clientFactory, tenant)
        check := CreateTenantCondition(tenant, test.check, test.name)
        err = waiter.WaitFor(t, check)
        assert.NilError(t, err)
        tenant, err = GetTenant(clientFactory, tenant)
        assert.NilError(t, err)
        tnn := tenant.Status.TenantNamespaceName
        t.Run("group", func(t *testing.T) {
                count := 0
                for _, testPlan := range testPlans {
                        count = count + testPlan.parallel
                        pipelineTest := testPlan.testBuilder(tnn)
                        for i := 1; i <= testPlan.parallel; i++ {
                                name :=
                                        fmt.Sprintf("%s_%d", pipelineTest.name, i)
                                log.Printf("Create Test: %s", name)
                                t.Run(name, func(t *testing.T) {
                                       pipelineTest := pipelineTest
                                       waiter := waiter
                                       name := name
                                       clientFactory := clientFactory 
                                            t.Parallel()
                                        pr, err := createPipelineRun(clientFactory, pipelineTest.pipelineRun)
                                        assert.NilError(t, err)
                                        log.Printf("pipeline run created for test: %s", name)
pipelineRunCheck := CreatePipelineRunCondition(pr, pipelineTest.check, name)
                                        err = waiter.WaitFor(t, pipelineRunCheck)
                                        assert.NilError(t, err)
                                })
                        }
                }
                log.Printf("###################")
                log.Printf("# Parallel: %d", count + 1)
                log.Printf("###################")

        })
}

