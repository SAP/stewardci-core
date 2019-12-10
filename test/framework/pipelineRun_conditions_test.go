package framework

import (
	"context"
	"fmt"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pipelineRun(name, namespace string) *api.PipelineRun {
	return &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func Test_CreatePipelineRunCondition(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		found          bool
		checkResult    bool
		checkError     error
		expectedResult bool
		expectedError  string
	}{
		{true, true, nil, true, ""},
		{true, false, nil, false, ""},
		{true, true, fmt.Errorf("foo"), true, "foo"},
		{false, true, fmt.Errorf("foo"), true, "pipelinerun not found .*"},
		{false, false, fmt.Errorf("foo"), true, "pipelinerun not found .*"},
	} {
		name := fmt.Sprintf("Found: %t, check: %t", test.found, test.checkResult)
		t.Run(name, func(t *testing.T) {
			// SETUP
			ctx := context.Background()
			pipelineRun := pipelineRun("foo", "bar")
			clientFactory := fake.NewClientFactory()
			if test.found {
				_, err := clientFactory.StewardV1alpha1().PipelineRuns("bar").Create(pipelineRun)
				assert.NilError(t, err, "Setup error")
			}
			ctx = SetClientFactory(ctx, clientFactory)
			check := func(*api.PipelineRun) (bool, error) {
				return test.checkResult, test.checkError
			}
			// EXERCISE
			condition := CreatePipelineRunCondition(pipelineRun, check)
			result, err := condition(ctx)
			// VERIFY
			if test.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, err != nil)
				assert.Assert(t, is.Regexp(test.expectedError, err.Error()))
			}
			assert.Assert(t, test.expectedResult == result)
		})
	}
}

func Test_PipelineRunCondition(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		examine        PipelineRunCheck
		pipelineStatus api.PipelineStatus
		expectedResult bool
		expectedError  string
	}{
		{name: "empty status",
			examine:        PipelineRunHasStateResult(api.ResultSuccess),
			pipelineStatus: api.PipelineStatus{},
			expectedResult: false,
			expectedError:  ""},
		{name: "success",
			examine:        PipelineRunHasStateResult(api.ResultSuccess),
			pipelineStatus: api.PipelineStatus{Result: api.ResultSuccess},
			expectedResult: true,
			expectedError:  ""},
		{name: "wrongStatus",
			examine:        PipelineRunHasStateResult(api.ResultSuccess),
			pipelineStatus: api.PipelineStatus{Result: api.ResultErrorInfra},
			expectedResult: true,
			expectedError:  `unexpected result: expecting "success", got "error_infra"`},
		{name: "undefined status",
			examine:        PipelineRunMessageOnFinished("foo"),
			pipelineStatus: api.PipelineStatus{},
			expectedResult: false,
			expectedError:  "",
		},
		{name: "correct message",
			examine: PipelineRunMessageOnFinished("foo"),
			pipelineStatus: api.PipelineStatus{State: api.StateFinished,
				Message: "foo"},
			expectedResult: true,
			expectedError:  "",
		},
		{name: "wrong message",
			examine: PipelineRunMessageOnFinished("foo"),
			pipelineStatus: api.PipelineStatus{State: api.StateFinished,
				Message: "bar"},
			expectedResult: true,
			expectedError:  `unexpected message: expecting "foo", got "bar"`,
		},
		{name: "empty message",
			examine:        PipelineRunMessageOnFinished("foo"),
			pipelineStatus: api.PipelineStatus{State: api.StateFinished},
			expectedResult: true,
			expectedError:  `unexpected message: expecting "foo", got ""`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			//SETUP
			mockPipelineRun := &api.PipelineRun{
				Status: test.pipelineStatus,
			}
			//EXERCISE
			result, err := test.examine(mockPipelineRun)
			// VERIFY
			if test.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, err != nil)
				assert.Assert(t, is.Regexp(test.expectedError, err.Error()))
			}
			assert.Assert(t, result == test.expectedResult)
		})
	}
}
