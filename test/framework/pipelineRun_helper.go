package framework

import (
	"context"
	"fmt"
	"log"
	"sync"
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
	var waitWG sync.WaitGroup
	for _, TestPlan := range TestPlans {
		waitWG.Add(TestPlan.Parallel)
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
				go func(waitWG *sync.WaitGroup) {
					myTestRun := createPipelineRunTest(pipelineTest, myTestRun)
					startWait(t, myTestRun, waitWG)
				}(&waitWG)
			} else {
				myTestRun := createPipelineRunTest(pipelineTest, myTestRun)
				go func(waitWG *sync.WaitGroup) {
					startWait(t, myTestRun, waitWG)
				}(&waitWG)
				time.Sleep(TestPlan.CreationDelay)
			}
		}
	}
	waitWG.Wait()
}

func checkResult(t *testing.T, run testRun) {
	info := fmt.Sprintf("Failing test: %q", run.Name)
	if run.Expected == "" {
		assert.NilError(t, run.result, info)
	} else {
		assert.Assert(t, is.Regexp(run.Expected, run.result.Error()), info)
	}
	log.Printf("Test %q completed", run.Name)
}

func startWait(t *testing.T, run testRun, waitWG *sync.WaitGroup) {
	defer func() {
		waitWG.Done()
	}()
	if run.result != nil {
		checkResult(t, run)
		return
	}
	ctx := run.ctx
	assert.NilError(t, ctx.Err(), fmt.Sprintf("Test: %q", run.Name))
	pr := GetPipelineRun(ctx)
	PipelineRunCheck := CreatePipelineRunCondition(pr, run.Check)
	err := WaitFor(ctx, PipelineRunCheck)
	run.result = err
	checkResult(t, run)
}

func createPipelineRunTest(pipelineTest PipelineRunTest, run testRun) testRun {

	PipelineRun := pipelineTest.PipelineRun
	ctx := run.ctx
	factory := GetClientFactory(ctx)
	Namespace := PipelineRun.GetNamespace()
	secretInterface := factory.CoreV1().Secrets(Namespace)
	for _, secret := range pipelineTest.Secrets {
		_, err := secretInterface.Create(secret)
		if err != nil {
			run.result = fmt.Errorf("secret creation failed: %q", err.Error())
			return run
		}
	}
	stewardClient := factory.StewardV1alpha1().PipelineRuns(Namespace)
	pr, err := stewardClient.Create(PipelineRun)
	if err != nil {
		run.result = fmt.Errorf("pipeline run creation failed: %q", err.Error())
		return run
	}
	log.Printf("pipeline run created for test: %s, %s/%s", run.Name, pr.GetNamespace(), pr.GetName())
	ctx = SetPipelineRun(ctx, pr)
	run.ctx = ctx
	return run
}

func setState(ctx context.Context, PipelineRun *api.PipelineRun, result api.Result) {
	fetcher := k8s.NewPipelineRunFetcher(GetClientFactory(ctx))
	pr, _ := fetcher.ByName(PipelineRun.GetNamespace(), PipelineRun.GetName())
	pr.UpdateResult(result)
}
