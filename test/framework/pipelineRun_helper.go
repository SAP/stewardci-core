package framework

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/google/uuid"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testRun struct {
	name     string
	ctx      context.Context
	check    PipelineRunCheck
	result   error
	expected string
	cleanup  bool
}

// ExecutePipelineRunTests execute a set of testPlans
func ExecutePipelineRunTests(t *testing.T, testPlans ...TestPlan) {
	executePipelineRunTests(Setup(t), t, testPlans...)
}

func executePipelineRunTests(ctx context.Context, t *testing.T, testPlans ...TestPlan) {
	tnn := GetNamespace(ctx)
	var waitWG sync.WaitGroup
	for _, testPlan := range testPlans {
		waitWG.Add(testPlan.Count)
		for i := 1; i <= testPlan.Count; i++ {
			name := fmt.Sprintf("%s_%d", getTestPlanName(testPlan), i)
			ctx = SetTestName(ctx, name)
			runID := &api.CustomJSON{
				Value: map[string]string{
					"jobId":   name,
					"buildId": uuid.New().String(),
					"realmId": GetRealmUUID(ctx),
				},
			}

			pipelineTest := testPlan.TestBuilder(tnn, runID)

			ctx, cancel := context.WithTimeout(ctx, pipelineTest.Timeout)
			defer cancel()
			t.Logf("Test %q: Started", name)
			myTestRun := testRun{
				name:     name,
				ctx:      ctx,
				check:    pipelineTest.Check,
				expected: pipelineTest.Expected,
				cleanup:  testPlan.Cleanup,
			}
			if testPlan.ParallelCreation {
				go func(waitWG *sync.WaitGroup) {
					myTestRun := createPipelineRunTest(t, pipelineTest, myTestRun)
					startWait(t, myTestRun, waitWG)
				}(&waitWG)
			} else {
				myTestRun := createPipelineRunTest(t, pipelineTest, myTestRun)
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
	ctx := run.ctx
	pr := GetPipelineRun(ctx)
	defer func() {
		if run.cleanup {
			t.Logf("Test %q: Deleting pipeline run '%s/%s'", run.name, pr.GetNamespace(), pr.GetName())
			err := DeletePipelineRun(ctx, pr)
			if err != nil {
				t.Logf("Test %q: Failed to clean up pipeline run '%s/%s': %s", run.name, pr.GetNamespace(), pr.GetName(), err)
			}
		}
		waitWG.Done()
	}()
	if run.result != nil {
		assert.NilError(t, checkResult(run), "Test: %q", run.name)
		return
	}

	assert.NilError(t, ctx.Err(), "Test: %q", run.name)
	PipelineRunCheck := CreatePipelineRunCondition(pr, run.check)
	duration, err := WaitFor(ctx, PipelineRunCheck)
	t.Logf("Test %q: Waited for %.2fs", run.name, duration.Seconds())
	run.result = err
	assert.NilError(t, checkResult(run), "Test: %q", run.name)
}

func createPipelineRunTest(t *testing.T, pipelineTest PipelineRunTest, run testRun) testRun {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		t.Logf("Test %q: Setup completed (took %.2fs)", run.name, duration.Seconds())
	}()
	PipelineRun := pipelineTest.PipelineRun
	ctx := run.ctx
	factory := GetClientFactory(ctx)
	Namespace := PipelineRun.GetNamespace()
	secretInterface := factory.CoreV1().Secrets(Namespace)
	for _, secret := range pipelineTest.Secrets {
		_, err := secretInterface.Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			run.result = fmt.Errorf("secret creation failed: %q", err.Error())
			return run
		}
	}
	stewardClient := factory.StewardV1alpha1().PipelineRuns(Namespace)
	pr, err := stewardClient.Create(ctx, PipelineRun, metav1.CreateOptions{})
	if err != nil {
		run.result = fmt.Errorf("pipeline run creation failed: %q", err.Error())
		return run
	}
	t.Logf("Test %q: Created pipeline run '%s/%s'", run.name, pr.GetNamespace(), pr.GetName())
	ctx = SetPipelineRun(ctx, pr)
	run.ctx = ctx
	return run
}

// CreatePipelineRunFromJSON creates a PipelineRun resource on a client
func CreatePipelineRunFromJSON(ctx context.Context, pipelineRunJSON string) (result *api.PipelineRun, err error) {
	return createPipelineRunFromString(ctx, pipelineRunJSON, "application/json")
}

// CreatePipelineRunFromYAML creates a PipelineRun resource on a client
func CreatePipelineRunFromYAML(ctx context.Context, pipelineRunYAML string) (result *api.PipelineRun, err error) {
	return createPipelineRunFromString(ctx, pipelineRunYAML, "application/yaml")
}

func createPipelineRunFromString(ctx context.Context, pipelineRunString string, contentType string) (result *api.PipelineRun, err error) {
	client := GetClientFactory(ctx).StewardV1alpha1().RESTClient()
	result = &api.PipelineRun{}
	err = client.Post().
		Namespace(GetNamespace(ctx)).
		Resource("pipelineruns").
		Body([]byte(pipelineRunString)).
		SetHeader("Content-Type", contentType).
		Do(ctx).
		Into(result)
	if err != nil {
		result = nil
	}
	return
}

// DeletePipelineRun deletes a PipelineRun resource from a client
func DeletePipelineRun(ctx context.Context, pipelineRun *api.PipelineRun) error {
	if pipelineRun == nil {
		return nil
	}
	stewardClient := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
	uid := pipelineRun.GetObjectMeta().GetUID()
	return stewardClient.Delete(ctx, pipelineRun.GetName(), metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &uid},
	})
}
