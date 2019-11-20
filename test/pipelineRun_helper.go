package test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"gotest.tools/assert"
)

type testRun struct {
	name  string
	ctx   context.Context
	check PipelineRunCheck
}

func executePipelineRunTests(t *testing.T, testPlans ...testPlan) {
	ctx := setup(t)
	test := TenantSuccessTest(ctx)
	tenant := test.tenant
	tenant, err := CreateTenant(ctx, tenant)
	assert.NilError(t, err)

	defer DeleteTenant(ctx, tenant)
	ctx = SetTestName(ctx, fmt.Sprintf("Create tenant for pipelineruns: %s", tenant.GetName()))
	check := CreateTenantCondition(tenant, test.check)
	err = WaitFor(ctx, check)
	assert.NilError(t, err)
	tenant, err = GetTenant(ctx, tenant)
	assert.NilError(t, err)
	tnn := tenant.Status.TenantNamespaceName
	count := 0
	for _, testPlan := range testPlans {
		count = count + testPlan.parallel
	}
	testChan := make(chan testRun, count)
	for _, testPlan := range testPlans {
		pipelineTest := testPlan.testBuilder(tnn)
		for i := 1; i <= testPlan.parallel; i++ {
			name :=
				fmt.Sprintf("%s_%d", pipelineTest.name, i)
			ctx = SetTestName(ctx, name)

			ctx, cancel := context.WithTimeout(ctx, pipelineTest.timeout)
			defer cancel()

			log.Printf("Create Test: %s", name)
			myTestRun := testRun{
				name:  name,
				ctx:   ctx,
				check: pipelineTest.check,
			}
			if testPlan.parallelCreation {
				go createPipelineRun(pipelineTest.pipelineRun, myTestRun, testChan)
			}
			if !testPlan.parallelCreation {
				single := make(chan testRun, 1)
				go createPipelineRun(pipelineTest.pipelineRun, myTestRun, single)
				time.Sleep(testPlan.creationDelay)
				x := <-single
				testChan <- x
			}
		}
	}
	resultChan := make(chan error, count)
	for i := count; i > 0; i-- {
		run := <-testChan
		ctx := run.ctx
		assert.NilError(t, ctx.Err())
		pr := GetPipelineRun(ctx)
		pipelineRunCheck := CreatePipelineRunCondition(pr, run.check)
		go func(pipelineRunCheck WaitConditionFunc) {
			err = WaitFor(ctx, pipelineRunCheck)
			resultChan <- err
		}(pipelineRunCheck)
	}
	for i := count; i > 0; i-- {
		log.Printf("Remaining: %d", i)
		err := <-resultChan
		assert.NilError(t, err)
	}
}

func createPipelineRun(pipelineRun *api.PipelineRun, run testRun, chanel chan testRun) {
	ctx := run.ctx
	stewardClient := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(pipelineRun.GetNamespace())
	pr, err := stewardClient.Create(pipelineRun)
	if err != nil {
		log.Printf("Creation failed: %s", err.Error())
		// todo return closed channel
	}
	log.Printf("pipeline run created for test: %s, %s/%s", run.name, pr.GetNamespace(), pr.GetName())
	ctx = SetPipelineRun(ctx, pr)
	run.ctx = ctx
	chanel <- run
}

func setState(ctx context.Context, pipelineRun *api.PipelineRun, result api.Result) {
	fetcher := k8s.NewPipelineRunFetcher(GetClientFactory(ctx))
	pr, _ := fetcher.ByName(pipelineRun.GetNamespace(), pipelineRun.GetName())
	pr.UpdateResult(result)
}
