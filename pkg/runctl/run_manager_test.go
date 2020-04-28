package runctl

import (
	"context"
	"testing"

	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	runi "github.com/SAP/stewardci-core/pkg/run"
	gomock "github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

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

	createTektonExecuted := false
	createRunNamespaceExecuted := false
	testing := &runInstanceTesting{
		createTektonTaskRunStub: func(ctx context.Context) error {
			createTektonExecuted = true
			return nil
		},
		prepareRunNamespaceStub: func(ctx context.Context) error {
			createRunNamespaceExecuted = true
			return nil
		},
	}
	ctx = WithRunInstanceTesting(ctx, testing)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	pipelineRun := mockPipelineRunWithNamespace(mockCtrl)
	// EXERCISE
	rm.Start(ctx, pipelineRun)

	// VERIFY
	assert.Assert(t, createTektonExecuted == true)
	assert.Assert(t, createRunNamespaceExecuted == true)
}

func createRunManagerAndContext() (runi.Manager, context.Context) {
	ctx := context.TODO()
	config := &pipelineRunsConfigStruct{}
	ctx = EnsureRunManager(ctx, config)
	rm := runi.GetRunManager(ctx)

	return rm, ctx
}

func mockPipelineRunWithNamespace(ctrl *gomock.Controller) *mocks.MockPipelineRun {
	runNamespace := "rn"
	mockPipelineRun := mocks.NewMockPipelineRun(ctrl)
	mockPipelineRun.EXPECT().GetRunNamespace().DoAndReturn(func() string {
		return runNamespace
	}).AnyTimes()
	return mockPipelineRun
}
