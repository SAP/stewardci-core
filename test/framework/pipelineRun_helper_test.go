package framework

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
			ctx := context.Background()
			factory := fake.NewClientFactory()
			ctx = SetClientFactory(ctx, factory)
			ctx = SetNamespace(ctx, "ns1")
			run := testRun{ctx: ctx}
			// EXERCISE
			testRun := createPipelineRunTest(test.test, run)
			//VALIDATE
			if test.expectedResult != "" {
				assert.Assert(t, testRun.result != nil)
				assert.Equal(t, testRun.result.Error(), test.expectedResult)
			} else {
				assert.NilError(t, testRun.result)
				secretInterface := factory.CoreV1().Secrets(GetNamespace(ctx))
				secretList, err := secretInterface.List(metav1.ListOptions{})
				log.Printf("Secrets: %+v", secretList)
				assert.NilError(t, err)

				assert.Equal(t, len(test.expectedSecretsByName), len(secretList.Items))
				for _, secretName := range test.expectedSecretsByName {
					secret, err := secretInterface.Get(secretName, metav1.GetOptions{})
					assert.NilError(t, err)
					assert.Equal(t, secret.GetName(), secretName)
				}
				pipelineRunInterface := factory.StewardV1alpha1().PipelineRuns(GetNamespace(ctx))
				pipelineRunName := GetPipelineRun(testRun.ctx).GetName()

				pipelineRun, err := pipelineRunInterface.Get(pipelineRunName, metav1.GetOptions{})
				assert.NilError(t, err)
				assert.Equal(t, pipelineRunName, pipelineRun.GetName())
			}
		})
	}
}
