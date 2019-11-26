package framework

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"
)

type testRun struct {
	name     string
	ctx      context.Context
	check    PipelineRunCheck
	result   error
	expected string
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
	_, err = WaitFor(ctx, Check)
	assert.NilError(t, err)
	tenant, err = GetTenant(ctx, tenant)
	assert.NilError(t, err)
	tnn := tenant.Status.TenantNamespaceName
	var waitWG sync.WaitGroup
	for _, TestPlan := range TestPlans {
		waitWG.Add(TestPlan.Parallel)
		pipelineTest := TestPlan.TestBuilder(tnn)
		for i := 1; i <= TestPlan.Parallel; i++ {
			name :=
				fmt.Sprintf("%s_%d", pipelineTest.Name, i)
			ctx = SetTestName(ctx, name)

			ctx, cancel := context.WithTimeout(ctx, pipelineTest.Timeout)
			defer cancel()

			log.Printf("Create Test: %s", name)
			myTestRun := testRun{
				name:     name,
				ctx:      ctx,
				check:    pipelineTest.Check,
				expected: pipelineTest.Expected,
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

func checkResult(run testRun) error {
	if run.expected == "" {
		if run.result != nil {
			return fmt.Errorf("unexpected error %q", run.result)
		}
	} else {
		pattern, err := regexp.Compile(run.expected)
		if err != nil {
			return fmt.Errorf("cannot compile expected %q", run.expected)
		}
		if !pattern.MatchString(run.result.Error()) {
			return fmt.Errorf("unexpected error, got %q expected %q", run.result.Error(), run.expected)
		}
	}
	return nil
}

func startWait(t *testing.T, run testRun, waitWG *sync.WaitGroup) {
	defer func() {
		waitWG.Done()
	}()
	if run.result != nil {
		assert.NilError(t, checkResult(run))
		return
	}
	ctx := run.ctx
	assert.NilError(t, ctx.Err(), fmt.Sprintf("Test: %q", run.name))
	pr := GetPipelineRun(ctx)
	PipelineRunCheck := CreatePipelineRunCondition(pr, run.check)
	duration, err := WaitFor(ctx, PipelineRunCheck)
	log.Printf("Test: %q waited for %s", run.name, duration)
	run.result = err
	assert.NilError(t, checkResult(run))
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
	log.Printf("pipeline run created for test: %s, %s/%s", run.name, pr.GetNamespace(), pr.GetName())
	ctx = SetPipelineRun(ctx, pr)
	run.ctx = ctx
	return run
}
