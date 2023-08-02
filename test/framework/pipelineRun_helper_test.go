package framework

import (
	"context"
	"fmt"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/pkg/utils"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupTestContext() context.Context {
	ctx := context.Background()
	ctx = SetNamespace(ctx, "ns1")
	factory := fake.NewClientFactory()
	return SetClientFactory(ctx, factory)
}

func pipelineWithStatusSuccess(namespace string, buildID *api.CustomJSON) PipelineRunTest {
	randomName, _ := utils.RandomAlphaNumString(5)
	pipelineRun := pipelineRun(randomName, namespace)
	pipelineRun.Status = api.PipelineStatus{Result: api.ResultSuccess}
	return PipelineRunTest{
		PipelineRun: pipelineRun,
		Check:       PipelineRunHasStateResult(api.ResultSuccess),
		Timeout:     time.Second,
	}
}

func Test_ExecutePipelineRunTests(t *testing.T) {
	// SETUP
	test := []TestPlan{
		{
			Name: "parallel5delay10mili",

			TestBuilder:   pipelineWithStatusSuccess,
			Count:         5,
			CreationDelay: time.Millisecond * 10,
		},
		{
			Name: "parallel5parallelcreation",

			TestBuilder:      pipelineWithStatusSuccess,
			Count:            5,
			ParallelCreation: true,
		},
		{
			Name: "parallel5nodelay",

			TestBuilder: pipelineWithStatusSuccess,
			Count:       5,
		},
	}
	ctx := setupTestContext()

	//EXERCISE
	executePipelineRunTests(ctx, t, test...)

	//VERIFY
	assert.NilError(t, ctx.Err())
	pipelineRunInterface := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
	pipelineRuns, err := pipelineRunInterface.List(ctx, metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 15, len(pipelineRuns.Items))
}

func Test_ExecutePipelineRunTestAndCleanupAfterwards(t *testing.T) {
	// SETUP
	test := []TestPlan{
		{
			Name: "parallelcreation",

			TestBuilder: pipelineWithStatusSuccess,
			Count:       1,
			Cleanup:     true,
		},
	}
	ctx := setupTestContext()

	//EXERCISE
	executePipelineRunTests(ctx, t, test...)

	//VERIFY
	assert.NilError(t, ctx.Err())
	pipelineRunInterface := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
	pipelineRuns, err := pipelineRunInterface.List(ctx, metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 0, len(pipelineRuns.Items))
}

func Test_ExecutePipelineRunTestAndDoNotCleanupAfterwards(t *testing.T) {
	// SETUP
	test := []TestPlan{
		{
			Name: "parallelcreationWithCleanupSetToFalse",

			TestBuilder: pipelineWithStatusSuccess,
			Count:       1,
			Cleanup:     false,
		},
		{
			Name: "parallelcreationWithoutCleanup",

			TestBuilder: pipelineWithStatusSuccess,
			Count:       1,
		},
	}
	ctx := setupTestContext()

	//EXERCISE
	executePipelineRunTests(ctx, t, test...)

	//VERIFY
	assert.NilError(t, ctx.Err())
	pipelineRunInterface := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
	pipelineRuns, err := pipelineRunInterface.List(ctx, metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 2, len(pipelineRuns.Items))
}

func Test_CheckResult(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		Run      testRun
		Expected error
	}{
		{
			Run:      testRun{expected: "", result: nil},
			Expected: nil,
		},
		{
			Run:      testRun{expected: "", result: fmt.Errorf("foo")},
			Expected: fmt.Errorf(`unexpected error "foo"`),
		},
		{
			Run:      testRun{expected: "foo", result: fmt.Errorf("foo")},
			Expected: nil,
		},
		{
			Run:      testRun{expected: "foo", result: fmt.Errorf("bar")},
			Expected: fmt.Errorf(`unexpected error, got "bar" expected "foo"`),
		},
		{
			Run:      testRun{expected: "bad(", result: fmt.Errorf("bar")},
			Expected: fmt.Errorf(`cannot compile expected "bad("`),
		},
	} {
		// EXERCISE
		error := checkResult(test.Run)

		// VERIFY
		if test.Expected == nil {
			assert.NilError(t, error)
		} else {
			assert.Assert(t, error != nil)
			assert.Equal(t, test.Expected.Error(), error.Error())
		}
	}
}

func Test_CreatePipelineRunTest(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		Name                  string
		Test                  PipelineRunTest
		ExpectedResult        string
		ExpectedSecretsByName []string
	}{
		{
			Name: "success",

			Test: PipelineRunTest{
				PipelineRun: pipelineRun("name1", "ns1"),
			},
			ExpectedResult:        "",
			ExpectedSecretsByName: []string{},
		},
		{
			Name: "secrets",

			Test: PipelineRunTest{
				PipelineRun: pipelineRun("name1", "ns1"),
				Secrets: []*v1.Secret{
					builder.SecretBasicAuth("foo", "ns1", "bar", "baz"),
					builder.SecretBasicAuth("bar", "ns1", "bar", "baz"),
				},
			},
			ExpectedResult:        "",
			ExpectedSecretsByName: []string{"foo", "bar"},
		},
		{
			Name: "duplicate secrets name",

			Test: PipelineRunTest{
				PipelineRun: pipelineRun("name1", "ns1"),
				Secrets: []*v1.Secret{
					builder.SecretBasicAuth("foo", "ns1", "bar", "baz"),
					builder.SecretBasicAuth("foo", "ns1", "bar", "baz"),
				},
			},
			ExpectedResult:        `secret creation failed: "secrets \"foo\" already exists"`,
			ExpectedSecretsByName: []string{"foo"},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()

			//SETUP
			ctx := setupTestContext()
			run := testRun{ctx: ctx}
			dummyT := &testing.T{}

			// EXERCISE
			testRun := createPipelineRunTest(dummyT, test.Test, run)

			//VERIFY
			if test.ExpectedResult != "" {
				assert.Assert(t, testRun.result != nil)
				assert.Equal(t, testRun.result.Error(), test.ExpectedResult)
			} else {
				assert.NilError(t, testRun.result)
				secretInterface := GetClientFactory(ctx).CoreV1().Secrets(GetNamespace(ctx))
				secretList, err := secretInterface.List(ctx, metav1.ListOptions{})
				assert.NilError(t, err)

				assert.Equal(t, len(test.ExpectedSecretsByName), len(secretList.Items))
				for _, secretName := range test.ExpectedSecretsByName {
					secret, err := secretInterface.Get(ctx, secretName, metav1.GetOptions{})
					assert.NilError(t, err)
					assert.Equal(t, secret.GetName(), secretName)
				}
				pipelineRunInterface := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
				pipelineRunName := GetPipelineRun(testRun.ctx).GetName()

				pipelineRun, err := pipelineRunInterface.Get(ctx, pipelineRunName, metav1.GetOptions{})
				assert.NilError(t, err)
				assert.Equal(t, pipelineRunName, pipelineRun.GetName())
			}
		})
	}
}
