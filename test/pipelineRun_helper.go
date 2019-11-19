package test

import (
	"fmt"
	"log"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
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
					if testPlan.parallelCreation {
						t.Parallel()
					}
					pr, err := createPipelineRun(clientFactory, pipelineTest.pipelineRun)
					assert.NilError(t, err)
					log.Printf("pipeline run created for test: %s", name)

					if !testPlan.parallelCreation {
						time.Sleep(testPlan.creationDelay)
						t.Parallel()
					}
					pipelineRunCheck := CreatePipelineRunCondition(pr, pipelineTest.check, name)
					err = waiter.WaitFor(t, pipelineRunCheck)
					assert.NilError(t, err)
				})
			}
		}
		log.Printf("###################")
		log.Printf("# Parallel: %d", count+1)
		log.Printf("###################")

	})
}

func setState(clientFactory k8s.ClientFactory, pipelineRun *api.PipelineRun, result api.Result) {
	fetcher := k8s.NewPipelineRunFetcher(clientFactory)
	pr, _ := fetcher.ByName(pipelineRun.GetNamespace(), pipelineRun.GetName())
	pr.UpdateResult(result)
}

func createPipelineRun(clientFactory k8s.ClientFactory, pipelineRun *api.PipelineRun) (*api.PipelineRun, error) {
	stewardClient := clientFactory.StewardV1alpha1().PipelineRuns(pipelineRun.GetNamespace())
	return stewardClient.Create(pipelineRun)
}
