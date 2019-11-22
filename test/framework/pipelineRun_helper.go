package framework

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

type testRun struct {
	Name     string
	ctx      context.Context
	Check    PipelineRunCheck
	result   error
	Expected string
}

// ExecutePipelineRunTests execute a set of TestPlans
func ExecutePipelineRunTests(t *testing.T, TestPlans ...TestPlan) {
	ctx := setup(t)
	test := TenantSuccessTest(ctx)
	tenant := test.tenant
	tenant, err := CreateTenant(ctx, tenant)
	assert.NilError(t, err)

	defer DeleteTenant(ctx, tenant)
	ctx = SetTestName(ctx, fmt.Sprintf("Create tenant for pipelineruns: %s", tenant.GetName()))
	Check := CreateTenantCondition(tenant, test.check)
	err = WaitFor(ctx, Check)
	assert.NilError(t, err)
	tenant, err = GetTenant(ctx, tenant)
	assert.NilError(t, err)
	tnn := tenant.Status.TenantNamespaceName
	count := 0
	for _, TestPlan := range TestPlans {
		count = count + TestPlan.Parallel
	}
	testChan := make(chan testRun, count)
	for _, TestPlan := range TestPlans {
		pipelineTest := TestPlan.TestBuilder(tnn)
		for i := 1; i <= TestPlan.Parallel; i++ {
			Name :=
				fmt.Sprintf("%s_%d", pipelineTest.Name, i)
			ctx = SetTestName(ctx, Name)

			ctx, cancel := context.WithTimeout(ctx, pipelineTest.Timeout)
			defer cancel()

			log.Printf("Create Test: %s", Name)
			myTestRun := testRun{
				Name:     Name,
				ctx:      ctx,
				Check:    pipelineTest.Check,
				Expected: pipelineTest.Expected,
			}
			if TestPlan.ParallelCreation {
				go createPipelineRunTest(pipelineTest, myTestRun, testChan)
			}
			if !TestPlan.ParallelCreation {
				single := make(chan testRun, 1)
				go createPipelineRunTest(pipelineTest, myTestRun, single)
				time.Sleep(TestPlan.CreationDelay)
				x := <-single
				testChan <- x
			}
		}
	}
	resultChan := make(chan testRun, count)
	for i := count; i > 0; i-- {
		run := <-testChan
		if run.result != nil {
			resultChan <- run
			log.Printf("Test %q completed", run.Name)
			continue
		}

		ctx := run.ctx
		assert.NilError(t, ctx.Err())
		pr := GetPipelineRun(ctx)
		PipelineRunCheck := CreatePipelineRunCondition(pr, run.Check)
		go func(PipelineRunCheck WaitConditionFunc) {
			err = WaitFor(ctx, PipelineRunCheck)
			run.result = err
			resultChan <- run
		}(PipelineRunCheck)
	}
	for i := count; i > 0; i-- {
		log.Printf("Remaining: %d", i)
		run := <-resultChan
		if run.Expected == "" {
			assert.NilError(t, run.result, fmt.Sprintf("Failing test: %q", run.Name))
		} else {
			assert.Assert(t, is.Regexp(run.Expected, run.result.Error()))
		}

	}
}

func createPipelineRunTest(pipelineTest PipelineRunTest, run testRun, chanel chan testRun) {

	PipelineRun := pipelineTest.PipelineRun
	ctx := run.ctx
	factory := GetClientFactory(ctx)
	Namespace := PipelineRun.GetNamespace()
	secretInterface := factory.CoreV1().Secrets(Namespace)
	for _, secret := range pipelineTest.Secrets {
		_, err := secretInterface.Create(secret)
		if err != nil {
			run.result = fmt.Errorf("secret creation failed: %q", err.Error())
			chanel <- run
			return
		}
	}
	stewardClient := factory.StewardV1alpha1().PipelineRuns(Namespace)
	pr, err := stewardClient.Create(PipelineRun)
	if err != nil {
		run.result = fmt.Errorf("pipeline run creation failed: %q", err.Error())
		chanel <- run
		return
	}
	log.Printf("pipeline run created for test: %s, %s/%s", run.Name, pr.GetNamespace(), pr.GetName())
	ctx = SetPipelineRun(ctx, pr)
	run.ctx = ctx
	chanel <- run
}

func setState(ctx context.Context, PipelineRun *api.PipelineRun, result api.Result) {
	fetcher := k8s.NewPipelineRunFetcher(GetClientFactory(ctx))
	pr, _ := fetcher.ByName(PipelineRun.GetNamespace(), PipelineRun.GetName())
	pr.UpdateResult(result)
}
