package framework

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	gomock "github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

func Test_PipelineRunHasStateResult_undefinedStatus(t *testing.T) {
	//SETUP
	examinee := PipelineRunHasStateResult(api.ResultSuccess)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockPipelineRun.EXPECT().GetStatus().Return(&api.PipelineStatus{}).AnyTimes()
	//EXERCISE
	result, err := examinee(mockPipelineRun)
	// VERIFY
	assert.NilError(t, err)
	assert.Assert(t, result == false)
}

func Test_PipelineRunHasStateResult_correctStatus(t *testing.T) {
	//SETUP
	examinee := PipelineRunHasStateResult(api.ResultSuccess)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	status := &api.PipelineStatus{Result: api.ResultSuccess}
	mockPipelineRun.EXPECT().GetStatus().Return(status).AnyTimes()
	//EXERCISE
	result, err := examinee(mockPipelineRun)
	// VERIFY
	assert.NilError(t, err)
	assert.Assert(t, result)
}

func Test_PipelineRunHasStateResult_wrongStatus(t *testing.T) {
	//SETUP
	examinee := PipelineRunHasStateResult(api.ResultSuccess)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	status := &api.PipelineStatus{Result: api.ResultErrorInfra}
	mockPipelineRun.EXPECT().GetStatus().Return(status).AnyTimes()
	//EXERCISE
	result, err := examinee(mockPipelineRun)
	// VERIFY
	assert.Equal(t, `unexpected result: expecting "success", got "error_infra"`, err.Error())
	assert.Assert(t, result)
}

func Test_PipelineRunMessageOnFinished_undefinedStatus(t *testing.T) {
	//SETUP
	examinee := PipelineRunMessageOnFinished("foo")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockPipelineRun.EXPECT().GetStatus().Return(&api.PipelineStatus{}).AnyTimes()
	//EXERCISE
	result, err := examinee(mockPipelineRun)
	// VERIFY
	assert.NilError(t, err)
	assert.Assert(t, result == false)
}

func Test_PipelineRunMessageOnFinished_correctMessage(t *testing.T) {
	//SETUP
	examinee := PipelineRunMessageOnFinished("foo")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	status := &api.PipelineStatus{State: api.StateFinished,
		Message: "foo"}
	mockPipelineRun.EXPECT().GetStatus().Return(status).AnyTimes()
	//EXERCISE
	result, err := examinee(mockPipelineRun)
	// VERIFY
	assert.NilError(t, err)
	assert.Assert(t, result)
}

func Test_PipelineRunMessageOnFinished_wrongMessage(t *testing.T) {
	//SETUP
	examinee := PipelineRunMessageOnFinished("foo")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	status := &api.PipelineStatus{State: api.StateFinished}
	mockPipelineRun.EXPECT().GetStatus().Return(status).AnyTimes()
	//EXERCISE
	result, err := examinee(mockPipelineRun)
	// VERIFY
	assert.Equal(t, `unexpected message: expecting "foo", got ""`, err.Error())
	assert.Assert(t, result)
}
