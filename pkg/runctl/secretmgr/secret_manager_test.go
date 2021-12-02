package secretmgr

import (
	"context"
	"fmt"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	gomock "github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

type testHelper struct {
	t                                *testing.T
	ctx                              context.Context
	pipelineSecretTransormerMatcher  gomock.Matcher
	imagePullSecretFilterMatcher     gomock.Matcher
	imagePullSecretTransormerMatcher gomock.Matcher
	cloneSecretTransormerMatcher     gomock.Matcher
	spec                             *stewardv1alpha1.PipelineSpec
}

func newTestHelper(t *testing.T) *testHelper {
	return &testHelper{
		t:                                t,
		ctx:                              context.Background(),
		pipelineSecretTransormerMatcher:  gomock.Len(2),
		imagePullSecretFilterMatcher:     gomock.Any(),
		imagePullSecretTransormerMatcher: gomock.Len(4),
		cloneSecretTransormerMatcher:     gomock.Len(4),

		spec: &stewardv1alpha1.PipelineSpec{
			JenkinsFile: stewardv1alpha1.JenkinsFile{
				RepoAuthSecret: "scm_secret1",
			},
			Secrets: []string{
				"secret1",
				"secret2",
			},
			ImagePullSecrets: []string{
				"imagePullSecret1",
				"imagePullSecret2",
			},
		},
	}
}

func mockPipelineRunWithSpec(th *testHelper) (*gomock.Controller, SecretManager, *mocks.MockPipelineRun, *secretMocks.MockSecretHelper) {
	mockCtrl := gomock.NewController(th.t)

	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	examinee := NewSecretManager(mockSecretHelper)

	// EXPECT
	mockPipelineRun.EXPECT().GetSpec().Return(th.spec).AnyTimes()
	mockPipelineRun.EXPECT().String().AnyTimes() //logging
	return mockCtrl, examinee, mockPipelineRun, mockSecretHelper
}

func Test_copyImagePullSecretsToRunNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	th := newTestHelper(t)
	mockCtrl, examinee, mockPipelineRun, mockSecretHelper := mockPipelineRunWithSpec(th)
	defer mockCtrl.Finish()

	// EXPECT
	mockSecretHelper.EXPECT().
		CopySecrets(
			th.ctx,
			[]string{"imagePullSecret1", "imagePullSecret2"},
			th.imagePullSecretFilterMatcher,
			th.imagePullSecretTransormerMatcher).
		Return([]string{"imagePullSecret1", "imagePullSecret2"}, nil)

	// EXERCISE
	names, err := examinee.copyImagePullSecretsToRunNamespace(th.ctx, mockPipelineRun)

	// VERIFY
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"imagePullSecret1", "imagePullSecret2"}, names)
}

func Test_copyPipelineCloneSecretToRunNamespace_Success(t *testing.T) {
	t.Parallel()

	// SETUP
	th := newTestHelper(t)
	mockCtrl, examinee, mockPipelineRun, mockSecretHelper := mockPipelineRunWithSpec(th)
	defer mockCtrl.Finish()

	// VERIFY
	mockPipelineRun.EXPECT().GetPipelineRepoServerURL().Return("server", nil).AnyTimes()
	mockSecretHelper.EXPECT().
		CopySecrets(th.ctx, []string{"scm_secret1"}, nil, th.cloneSecretTransormerMatcher).
		Return([]string{"scm_secret1"}, nil)

	// EXERCISE
	examinee.copyPipelineCloneSecretToRunNamespace(th.ctx, mockPipelineRun)
}

func Test_copyPipelineCloneSecretToRunNamespace_FailsWithContentErrorOnGetPipelineRepoServerURLError(t *testing.T) {
	t.Parallel()

	// SETUP
	th := newTestHelper(t)
	mockCtrl, examinee, mockPipelineRun, _ := mockPipelineRunWithSpec(th)
	defer mockCtrl.Finish()

	// EXPECT
	mockPipelineRun.EXPECT().GetPipelineRepoServerURL().Return("", fmt.Errorf("err1")).AnyTimes()

	// EXERCISE
	_, err := examinee.copyPipelineCloneSecretToRunNamespace(th.ctx, mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Equal(t, "err1", err.Error())
	assert.Equal(t, stewardv1alpha1.ResultErrorContent, serrors.GetClass(err))
}

func Test_copyPipelineSecretsToRunNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	th := newTestHelper(t)
	mockCtrl, examinee, mockPipelineRun, mockSecretHelper := mockPipelineRunWithSpec(th)
	defer mockCtrl.Finish()

	// VERIFY
	mockSecretHelper.EXPECT().
		CopySecrets(th.ctx, []string{"secret1", "secret2"}, nil, th.pipelineSecretTransormerMatcher).
		Return([]string{"secret1", "secret2"}, nil)

	// EXERCISE
	examinee.copyPipelineSecretsToRunNamespace(th.ctx, mockPipelineRun)

}

func Test_copySecrets_FailsWithContentErrorOnNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	th := newTestHelper(t)
	mockCtrl, examinee, mockPipelineRun, mockSecretHelper := mockPipelineRunWithSpec(th)
	defer mockCtrl.Finish()

	expectedError := fmt.Errorf("err1")
	// EXPECT
	mockSecretHelper.EXPECT().
		CopySecrets(th.ctx, []string{"foo"}, nil, nil).Return(nil, expectedError)
	mockSecretHelper.EXPECT().
		IsNotFound(expectedError).Return(true)

	// EXERCISE
	_, err := examinee.copySecrets(th.ctx, mockPipelineRun, []string{"foo"}, nil, nil)

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "err1", err.Error())
	assert.Equal(t, stewardv1alpha1.ResultErrorContent, serrors.GetClass(err))
}

func Test_copySecrets_FailsWithInfraErrorOnOtherError(t *testing.T) {
	t.Parallel()

	// SETUP
	th := newTestHelper(t)
	mockCtrl, examinee, mockPipelineRun, mockSecretHelper := mockPipelineRunWithSpec(th)
	defer mockCtrl.Finish()

	expectedError := fmt.Errorf("err1")
	// EXPECT
	mockSecretHelper.EXPECT().
		CopySecrets(th.ctx, []string{"foo"}, nil, nil).Return(nil, expectedError)
	mockSecretHelper.EXPECT().
		IsNotFound(expectedError).Return(false)

	// EXERCISE
	_, err := examinee.copySecrets(th.ctx, mockPipelineRun, []string{"foo"}, nil, nil)

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "err1", err.Error())
	assert.Equal(t, stewardv1alpha1.ResultErrorInfra, serrors.GetClass(err))
}
