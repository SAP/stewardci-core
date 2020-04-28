package runctl

import (
	"context"
	"fmt"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	runi "github.com/SAP/stewardci-core/pkg/run"
	gomock "github.com/golang/mock/gomock"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const runNamespaceName = "rn"

func Test_EnsureRunManager_CreateIfMissing(t *testing.T) {
	t.Parallel()
	// SETUP
	ctx := context.TODO()

	assert.Assert(t, runi.GetRunManager(ctx) == nil)

	config := &pipelineRunsConfigStruct{}

	// EXERCISE
	ctx = EnsureRunManager(ctx, config)

	// VERIFY
	assert.DeepEqual(t, config,
		runi.GetRunManager(ctx).(*runManager).pipelineRunsConfig)
}

func Test_EnsureRunManager_DontModifyIfExists(t *testing.T) {
	t.Parallel()
	// SETUP
	ctx := context.TODO()
	config := &pipelineRunsConfigStruct{}
	ctx = EnsureRunManager(ctx, config)

	// EXERCISE
	ctxnew := EnsureRunManager(ctx, config)

	// VERIFY
	assert.Assert(t, ctx == ctxnew)
}

func Test_StartRunManager(t *testing.T) {
	t.Parallel()
	// SETUP
	rm, ctx := createRunManagerAndContext()
	tektonError := fmt.Errorf("tekton")
	namespaceError := fmt.Errorf("namespace")

	for _, test := range []struct {
		name                         string
		createRunNamespaceError      error
		createTektonTaskError        error
		expectedRunNamespaceExecuted bool
		expectedTektonTaskExecuted   bool
		expectedError                error
	}{
		{"ok",
			nil, nil,
			true, true,
			nil,
		},
		{"tekton error",
			nil, tektonError,
			true, true,
			tektonError,
		},
		{"namespace error",
			namespaceError, nil,
			true, false,
			namespaceError,
		},
		{"both error",
			namespaceError, tektonError,
			true, false, namespaceError},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()
			// SETUP

			createTektonExecuted := false
			createRunNamespaceExecuted := false
			testing := &runInstanceTesting{
				createTektonTaskRunStub: func(ctx context.Context) error {
					createTektonExecuted = true
					return test.createTektonTaskError
				},
				prepareRunNamespaceStub: func(ctx context.Context) error {
					createRunNamespaceExecuted = true
					return test.createRunNamespaceError
				},
			}
			ctx = WithRunInstanceTesting(ctx, testing)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			pipelineRun := mockPipelineRunWithNamespace(mockCtrl)
			// EXERCISE
			err := rm.Start(ctx, pipelineRun)
			// VERIFY
			if test.expectedError == nil {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, test.expectedError == err)
				assert.Assert(t, createTektonExecuted == test.expectedTektonTaskExecuted)
				assert.Assert(t, createRunNamespaceExecuted == test.expectedRunNamespaceExecuted)
			}
		})
	}
}

func Test_GetRun_missing(t *testing.T) {
	t.Parallel()
	// SETUP
	rm, ctx := createRunManagerAndContext()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx = mockFactories(ctx, mockCtrl)
	pipelineRun := mockPipelineRunWithNamespace(mockCtrl)
	// EXERCISE
	_, err := rm.GetRun(ctx, pipelineRun)
	assert.Assert(t, err != nil)
}

func Test_GetRun_found(t *testing.T) {
	t.Parallel()
	// SETUP
	rm, ctx := createRunManagerAndContext()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
pipelineRun := mockPipelineRunWithNamespace(mockCtrl)
	tektonTaskRun := tekton.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tektonTaskRunName,
			Namespace: runNamespaceName,
		},
	}
	cf := fake.NewClientFactory()
        _,err := cf.TektonV1alpha1().TaskRuns(runNamespaceName).Create(&tektonTaskRun)
        assert.NilError(t,err)	
        ctx = k8s.WithClientFactory(ctx, cf)
	// EXERCISE
	run, err := rm.GetRun(ctx, pipelineRun)
	assert.NilError(t, err)
	assert.Assert(t, run != nil)
}

func Test_CleanupRunManager(t *testing.T) {
	t.Parallel()
	// SETUP
	rm, ctx := createRunManagerAndContext()
	cleanupError := fmt.Errorf("cleanup")

	for _, test := range []struct {
		name          string
		cleanupError  error
		expectedError error
	}{
		{"ok", nil, nil},
		{"error", cleanupError, cleanupError},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()
			// SETUP

			executed := false
			testing := &runInstanceTesting{
				cleanupStub: func(ctx context.Context) error {
					executed = true
					return test.cleanupError
				},
			}
			ctx = WithRunInstanceTesting(ctx, testing)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			pipelineRun := mockPipelineRunWithNamespace(mockCtrl)
			// EXERCISE
			err := rm.Cleanup(ctx, pipelineRun)
			// VERIFY
			if test.expectedError == nil {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, test.expectedError == err)
				assert.Assert(t, executed)
			}
		})
	}

}

func createRunManagerAndContext() (runi.Manager, context.Context) {
	ctx := context.TODO()
	config := &pipelineRunsConfigStruct{}
	ctx = EnsureRunManager(ctx, config)
	rm := runi.GetRunManager(ctx)

	return rm, ctx
}

func mockPipelineRunWithNamespace(ctrl *gomock.Controller) *mocks.MockPipelineRun {
	mockPipelineRun := mocks.NewMockPipelineRun(ctrl)
	mockPipelineRun.EXPECT().GetRunNamespace().DoAndReturn(func() string {
		return runNamespaceName
	}).AnyTimes()
	return mockPipelineRun
}
