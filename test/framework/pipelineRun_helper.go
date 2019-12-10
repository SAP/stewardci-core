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

// ExecutePipelineRunTests execute a set of testPlans
func ExecutePipelineRunTests(t *testing.T, testPlans ...TestPlan) {
	executePipelineRunTests(setup(t), t, testPlans...)
}

func executePipelineRunTests(ctx context.Context, t *testing.T, testPlans ...TestPlan) {
	rollback, ctx := ensureTenant(ctx, t)
	defer rollback()
	tnn := GetTenantNamespace(ctx)
	var waitWG sync.WaitGroup
	for _, testPlan := range testPlans {
		waitWG.Add(testPlan.Count)
		for i := 1; i <= testPlan.Count; i++ {
			name :=
				fmt.Sprintf("%s_%d", getTestPlanName(testPlan), i)
			ctx = SetTestName(ctx, name)
			pipelineTest := testPlan.TestBuilder(tnn)

			ctx, cancel := context.WithTimeout(ctx, pipelineTest.Timeout)
			defer cancel()
			log.Printf("Test: %q start", name)
			myTestRun := testRun{
				name:     name,
				ctx:      ctx,
				check:    pipelineTest.Check,
				expected: pipelineTest.Expected,
			}
			if testPlan.ParallelCreation {
				go func(waitWG *sync.WaitGroup) {
					myTestRun := createPipelineRunTest(pipelineTest, myTestRun)
					startWait(t, myTestRun, waitWG)
				}(&waitWG)
			} else {
				myTestRun := createPipelineRunTest(pipelineTest, myTestRun)
				go func(waitWG *sync.WaitGroup) {
					startWait(t, myTestRun, waitWG)
				}(&waitWG)
				time.Sleep(testPlan.CreationDelay)
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
	log.Printf("Test: %q waited for %.2f s", run.name, duration.Seconds())
	run.result = err
	assert.NilError(t, checkResult(run), fmt.Sprintf("Test: %q", run.name))
}

func createPipelineRunTest(pipelineTest PipelineRunTest, run testRun) testRun {
	startTime := time.Now()
	defer func() {
		duration := time.Now().Sub(startTime)
		log.Printf("Test: %q setup took %.2f s", run.name, duration.Seconds())
	}()
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
	log.Printf("Test: %q pipeline run created '%s/%s'", run.name, pr.GetNamespace(), pr.GetName())
	ctx = SetPipelineRun(ctx, pr)
	run.ctx = ctx
	return run
}
