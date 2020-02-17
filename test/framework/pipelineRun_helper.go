package framework

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/google/uuid"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	executePipelineRunTests(Setup(t), t, testPlans...)
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
			runID := &api.CustomJSON{
				map[string]string{
					"jobId":    name,
					"biuildId": uuid.New().String(),
					"realmId":  GetRealmUUID(ctx),
				}}

			pipelineTest := testPlan.TestBuilder(tnn, runID)

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
		assert.NilError(t, checkResult(run), "Test: %q", run.name)
		return
	}
	ctx := run.ctx
	assert.NilError(t, ctx.Err(), "Test: %q", run.name)
	pr := GetPipelineRun(ctx)
	PipelineRunCheck := CreatePipelineRunCondition(pr, run.check)
	duration, err := WaitFor(ctx, PipelineRunCheck)
	log.Printf("Test: %q waited for %.2f s", run.name, duration.Seconds())
	run.result = err
	assert.NilError(t, checkResult(run), "Test: %q", run.name)
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

// CreatePipelineRunFromJSON creates a Tenant resource on a client
func CreatePipelineRunFromJSON(ctx context.Context, pipelineRunJSON string) (result *api.PipelineRun, err error) {
	return createPipelineRunFromString(ctx, pipelineRunJSON, "application/json")
}

// CreatePipelineRunFromYAML creates a Tenant resource on a client
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
		Do().
		Into(result)
	if err != nil {
		result = nil
	}
	return
}

// DeletePipelineRun deletes a Tenant resource from a client
func DeletePipelineRun(ctx context.Context, pipelineRun *api.PipelineRun) error {
	if pipelineRun == nil {
		return nil
	}
	stewardClient := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
	uid := pipelineRun.GetObjectMeta().GetUID()
	return stewardClient.Delete(pipelineRun.GetName(), &metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &uid},
	})
}
