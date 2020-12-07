package secretmgr

import (
	"fmt"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	gomock "github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

// TODO: write better matcher to check correct filters and transformers
var (
	pipelineSecretTransormerMatcher  = gomock.Len(2)
	imagePullSecretFilterMatcher     = gomock.Any()
	imagePullSecretTransormerMatcher = gomock.Len(4)
	cloneSecretTransormerMatcher     = gomock.Len(4)

	spec = &stewardv1alpha1.PipelineSpec{
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
	}
)

func Test_copyImagePullSecretsToRunNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	examinee := NewSecretManager(mockSecretHelper)

	// EXPECT
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"imagePullSecret1", "imagePullSecret2"},
			imagePullSecretFilterMatcher,
			imagePullSecretTransormerMatcher).
		Return([]string{"imagePullSecret1", "imagePullSecret2"}, nil)

	// EXERCISE
	names, err := examinee.copyImagePullSecretsToRunNamespace(mockPipelineRun)

	// VERIFY
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"imagePullSecret1", "imagePullSecret2"}, names)
}

func Test_copyPipelineCloneSecretToRunNamespace_Success(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	examinee := NewSecretManager(mockSecretHelper)

	// VERIFY
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockPipelineRun.EXPECT().GetPipelineRepoServerURL().Return("server", nil).AnyTimes()
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"scm_secret1"}, nil, cloneSecretTransormerMatcher).
		Return([]string{"scm_secret1"}, nil)

	// EXERCISE
	examinee.copyPipelineCloneSecretToRunNamespace(mockPipelineRun)
}

func Test_copyPipelineCloneSecretToRunNamespace_FailsWithContentErrorOnGetPipelineRepoServerURLError(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	examinee := NewSecretManager(mockSecretHelper)

	// EXPECT
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockPipelineRun.EXPECT().GetPipelineRepoServerURL().Return("", fmt.Errorf("err1")).AnyTimes()

	// EXERCISE
	_, err := examinee.copyPipelineCloneSecretToRunNamespace(mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Equal(t, "err1", err.Error())
	assert.Equal(t, stewardv1alpha1.ResultErrorContent, serrors.GetClass(err))
}

func Test_copyPipelineSecretsToRunNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	examinee := NewSecretManager(mockSecretHelper)

	// VERIFY
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"secret1", "secret2"}, nil, pipelineSecretTransormerMatcher).
		Return([]string{"secret1", "secret2"}, nil)

	// EXERCISE
	examinee.copyPipelineSecretsToRunNamespace(mockPipelineRun)

}

func Test_copySecrets_FailsWithContentErrorOnNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	examinee := NewSecretManager(mockSecretHelper)
	expectedError := fmt.Errorf("err1")
	// EXPECT
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"foo"}, nil, nil).Return(nil, expectedError)
	mockSecretHelper.EXPECT().
		IsNotFound(expectedError).Return(true)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	_, err := examinee.copySecrets(mockPipelineRun, []string{"foo"}, nil, nil)

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "err1", err.Error())
	assert.Equal(t, stewardv1alpha1.ResultErrorContent, serrors.GetClass(err))
}

func Test_copySecrets_FailsWithInfraErrorOnOtherError(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPipelineRun := mocks.NewMockPipelineRun(mockCtrl)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	examinee := NewSecretManager(mockSecretHelper)
	expectedError := fmt.Errorf("err1")
	// EXPECT
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"foo"}, nil, nil).Return(nil, expectedError)
	mockSecretHelper.EXPECT().
		IsNotFound(expectedError).Return(false)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	_, err := examinee.copySecrets(mockPipelineRun, []string{"foo"}, nil, nil)

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "err1", err.Error())
	assert.Equal(t, stewardv1alpha1.ResultErrorInfra, serrors.GetClass(err))
}
