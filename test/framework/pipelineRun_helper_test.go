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
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupTestContext() context.Context {
	ctx := context.Background()
	ctx = SetNamespace(ctx, "ns1")
	factory := fake.NewClientFactory()
	ctx = SetTenantNamespace(ctx, "ns1")
	return SetClientFactory(ctx, factory)
}

func pipelineWithStatusSuccess(namespace string) PipelineRunTest {
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
		TestPlan{Name: "parallel5delay10mili",
			TestBuilder:   pipelineWithStatusSuccess,
			Count:         5,
			CreationDelay: time.Millisecond * 10,
		},
		TestPlan{Name: "parallel5parallelcreation",
			TestBuilder:      pipelineWithStatusSuccess,
			Count:            5,
			ParallelCreation: true,
		},
		TestPlan{Name: "parallel5nodelay",
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
	pipelineRuns, err := pipelineRunInterface.List(metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 15, len(pipelineRuns.Items))
}

func Test_CheckResult(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		run      testRun
		expected error
	}{
		{testRun{expected: "", result: nil}, nil},
		{testRun{expected: "", result: fmt.Errorf("foo")}, fmt.Errorf(`unexpected error "foo"`)},
		{testRun{expected: "foo", result: fmt.Errorf("foo")}, nil},
		{testRun{expected: "foo", result: fmt.Errorf("bar")},
			fmt.Errorf(`unexpected error, got "bar" expected "foo"`)},
		{testRun{expected: "bad(", result: fmt.Errorf("bar")},
			fmt.Errorf(`cannot compile expected "bad("`)},
	} {
		// EXERCISE
		error := checkResult(test.run)
		// VERIFY
		if test.expected == nil {
			assert.NilError(t, error)
		} else {
			assert.Assert(t, error != nil)
			assert.Equal(t, test.expected.Error(), error.Error())
		}
	}
}

func Test_CreatePipelineRunTest(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                  string
		test                  PipelineRunTest
		expectedResult        string
		expectedSecretsByName []string
	}{
		{
			name:                  "success",
			test:                  PipelineRunTest{PipelineRun: pipelineRun("name1", "ns1")},
			expectedResult:        "",
			expectedSecretsByName: []string{},
		}, {

			name: "secrets",
			test: PipelineRunTest{PipelineRun: pipelineRun("name1", "ns1"),
				Secrets: []*v1.Secret{builder.SecretBasicAuth("foo", "ns1", "bar", "baz"),
					builder.SecretBasicAuth("bar", "ns1", "bar", "baz")},
			},
			expectedResult:        "",
			expectedSecretsByName: []string{"foo", "bar"},
		}, {
			name: "duplicate secrets name",
			test: PipelineRunTest{PipelineRun: pipelineRun("name1", "ns1"),
				Secrets: []*v1.Secret{builder.SecretBasicAuth("foo", "ns1", "bar", "baz"),
					builder.SecretBasicAuth("foo", "ns1", "bar", "baz")},
			},
			expectedResult:        `secret creation failed: "secrets \"foo\" already exists"`,
			expectedSecretsByName: []string{"foo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()
			//SETUP
			ctx := setupTestContext()
			run := testRun{ctx: ctx}
			// EXERCISE
			testRun := createPipelineRunTest(test.test, run)
			//VERIFY
			if test.expectedResult != "" {
				assert.Assert(t, testRun.result != nil)
				assert.Equal(t, testRun.result.Error(), test.expectedResult)
			} else {
				assert.NilError(t, testRun.result)
				secretInterface := GetClientFactory(ctx).CoreV1().Secrets(GetNamespace(ctx))
				secretList, err := secretInterface.List(metav1.ListOptions{})
				assert.NilError(t, err)

				assert.Equal(t, len(test.expectedSecretsByName), len(secretList.Items))
				for _, secretName := range test.expectedSecretsByName {
					secret, err := secretInterface.Get(secretName, metav1.GetOptions{})
					assert.NilError(t, err)
					assert.Equal(t, secret.GetName(), secretName)
				}
				pipelineRunInterface := GetClientFactory(ctx).StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
				pipelineRunName := GetPipelineRun(testRun.ctx).GetName()

				pipelineRun, err := pipelineRunInterface.Get(pipelineRunName, metav1.GetOptions{})
				assert.NilError(t, err)
				assert.Equal(t, pipelineRunName, pipelineRun.GetName())
			}
		})
	}
}
