package runctl

// This tests cross the border between runManager and runInstance
// More refactroing required

import (
	"context"
	"fmt"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	k8sfake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	runi "github.com/SAP/stewardci-core/pkg/run"
	"github.com/davecgh/go-spew/spew"
	gomock "github.com/golang/mock/gomock"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func Test_RunManager_StartPipelineRun_DoesNotSetPipelineRunStatus(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := mockContext(mockCtrl)
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, nil)
	preparePredefinedClusterRole(ctx, t)
	config := &pipelineRunsConfigStruct{}
	ctx = EnsureRunManager(ctx, config)
	examinee := runi.GetRunManager(ctx)
	// EXERCISE
	err := examinee.Start(ctx, mockPipelineRun)
	assert.NilError(t, err)

	// VERIFY
	// UpdateState should never be called
	mockPipelineRun.EXPECT().UpdateState(gomock.Any()).Times(0)
}

func Test_RunManager_StartPipelineRun_DoesCopySecret(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	spec := &stewardv1alpha1.PipelineSpec{
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
	ctx := mockContext(mockCtrl)
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)
	// UpdateState should never be called
	mockPipelineRun.EXPECT().
		UpdateState(gomock.Any()).
		Do(func(interface{}) { panic("unexpected call") }).
		AnyTimes()

	preparePredefinedClusterRole(ctx, t)
	config := &pipelineRunsConfigStruct{}

	// inject secret helper mock
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	testing := newRunManagerTestingWithRequiredStubs()
	testing.getSecretHelperStub = func(string, corev1.SecretInterface) secrets.SecretHelper {
		return mockSecretHelper
	}
	ctx = withRunInstanceTesting(ctx, testing)
	ctx = EnsureRunManager(ctx, config)
	examinee := runi.GetRunManager(ctx)
	// EXPECT
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"scm_secret1"}, nil, gomock.Any()).
		Return([]string{"scm_secret1"}, nil)
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"secret1", "secret2"}, nil, gomock.Any()).
		Return([]string{"secret1", "secret2"}, nil)
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"imagePullSecret1", "imagePullSecret2"}, gomock.Any(), gomock.Any()).
		Return([]string{"imagePullSecret1", "imagePullSecret2"}, nil)

	// EXERCISE
	err := examinee.Start(ctx, mockPipelineRun)
	assert.NilError(t, err)
}

func Test_RunManager_Start_FailsWithContentErrorWhenPipelineCloneSecretNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		JenkinsFile: stewardv1alpha1.JenkinsFile{
			RepoAuthSecret: secretName,
		},
	}
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)
	ctx := mockContext(mockCtrl)
	preparePredefinedClusterRole(ctx, t)
	config := &pipelineRunsConfigStruct{}
	ctx = withRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())
	ctx = EnsureRunManager(ctx, config)
	examinee := runi.GetRunManager(ctx)
	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)

	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(ctx, mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Assert(t, is.Regexp("failed to copy pipeline clone secret: .*", err.Error()))
}

func Test_RunManager_Start_FailsWithContentErrorWhenSecretNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		Secrets: []string{secretName},
	}
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)
	config := &pipelineRunsConfigStruct{}
	ctx := withRunInstanceTesting(mockContext(mockCtrl), newRunManagerTestingWithRequiredStubs())
	preparePredefinedClusterRole(ctx, t)
	ctx = EnsureRunManager(ctx, config)
	examinee := runi.GetRunManager(ctx)
	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)

	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(ctx, mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Assert(t, is.Regexp("failed to copy pipeline secrets: .*", err.Error()))
}

func Test_RunManager_Start_FailsWithContentErrorWhenImagePullSecretNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		ImagePullSecrets: []string{secretName},
	}
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)

	config := &pipelineRunsConfigStruct{}
	ctx := withRunInstanceTesting(mockContext(mockCtrl), newRunManagerTestingWithRequiredStubs())
	preparePredefinedClusterRole(ctx, t)
	ctx = EnsureRunManager(ctx, config)
	examinee := runi.GetRunManager(ctx)
	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)
	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(ctx, mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Assert(t, is.Regexp("failed to copy image pull secrets: .*", err.Error()))
}

func Test_RunManager_Start_FailsWithInfraErrorWhenForbidden(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "scm_secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		JenkinsFile: stewardv1alpha1.JenkinsFile{
			RepoAuthSecret: secretName,
		},
	}
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)

	config := &pipelineRunsConfigStruct{}
	ctx := mockContext(mockCtrl)
	ctx = withRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())
	preparePredefinedClusterRole(ctx, t)
	ctx = EnsureRunManager(ctx, config)
	examinee := runi.GetRunManager(ctx)
	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, fmt.Errorf("Forbidden"))
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)
	mockPipelineRun.EXPECT().UpdateMessage("Forbidden")
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorInfra)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(ctx, mockPipelineRun)
	assert.Assert(t, err != nil)
}

func Test_RunManager_Log_Elasticsearch(t *testing.T) {
	t.Parallel()

	const (
		TaskRunParamNameIndexURL  = "PIPELINE_LOG_ELASTICSEARCH_INDEX_URL"
		TaskRunParamNameRunIDJSON = "PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON"
	)

	findTaskRunParam := func(taskRun *tekton.TaskRun, paramName string) (param *tekton.Param) {
		assert.Assert(t, taskRun.Spec.Inputs.Params != nil)
		for _, p := range taskRun.Spec.Inputs.Params {
			if p.Name == paramName {
				if param != nil {
					t.Fatalf("input param specified twice: %s", paramName)
				}
				param = &p
			}
		}
		return
	}

	setupExaminee := func(
		ctx context.Context, t *testing.T, pipelineRunJSON string,
	) (
		ctxOut context.Context, runInstanceObj *runInstance, cf *k8sfake.ClientFactory,
	) {
		pipelineRun := StewardObjectFromJSON(t, pipelineRunJSON).(*stewardv1alpha1.PipelineRun)
		t.Log("decoded:\n", spew.Sdump(pipelineRun))

		cf = k8sfake.NewClientFactory(
			k8sfake.Namespace("namespace1"),
			pipelineRun,
		)
		ctx = k8s.WithClientFactory(ctx, cf)
		k8sPipelineRun, err := k8s.NewPipelineRun(pipelineRun, cf)
		assert.NilError(t, err)
		ctx = secrets.WithSecretProvider(ctx, k8s.NewTenantNamespace(cf, pipelineRun.GetNamespace()).GetSecretProvider())
		ctx = withRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())
		ctxOut = k8s.WithNamespaceManager(ctx, k8s.NewNamespaceManager(cf, "prefix1", 0))
		runInstanceObj = &runInstance{
			pipelineRun: k8sPipelineRun,
		}
		return
	}

	expectSingleTaskRun := func(t *testing.T, cf *k8sfake.ClientFactory, k8sPipelineRun k8s.PipelineRun) *tekton.TaskRun {
		taskRunList, err := cf.TektonV1alpha1().TaskRuns(k8sPipelineRun.GetRunNamespace()).List(metav1.ListOptions{})
		assert.NilError(t, err)
		assert.Equal(t, 1, len(taskRunList.Items), "%s", spew.Sdump(taskRunList))
		return &taskRunList.Items[0]
	}

	/**
	 * Test: Various JSON values for spec.logging.elasticsearch.runID
	 * are correctly passed as Tekton TaskRun input parameter.
	 */
	test := "Passthrough"
	for _, tc := range []struct {
		name               string
		runIDJSON          string
		expectedParamValue string
	}{
		{"none", ``, `null`},
		{"none2", `"___dummy___": 1`, `null`},
		{"null", `"runID": null`, `null`},
		{"true", `"runID": true`, `true`},
		{"false", `"runID": false`, `false`},
		{"int", `"runID": 123`, `123`},
		{"intneg", `"runID": -123`, `-123`},
		{"float", `"runID": 123.45`, `123.45`},
		{"floatneg", `"runID": -123.45`, `-123.45`},
		{"string", `"runID": "some string"`, `"some string"`},
		{"map", `"runID": { "key2": "value2", "key1": "value1" }`, `{"key1":"value1","key2":"value2"}`},
		{"mapdeep", `
			"runID": {
				"key1": {
					"key2": {
						"key3_1": "value3",
						"key3_2": null,
						"key3_3": [1, "2", true]
					}
				}
			}`,
			`{"key1":{"key2":{"key3_1":"value3","key3_2":null,"key3_3":[1,"2",true]}}}`},
	} {
		t.Run(test+"_"+tc.name, func(t *testing.T) {
			var err error

			// setup
			pipelineRunJSON := fmt.Sprintf(fixIndent(`
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "PipelineRun",
					"metadata": {
						"name": "dummy1",
						"namespace": "namespace1"
					},
					"spec": {
						"jenkinsFile": {
							"repoUrl": "dummyRepoUrl",
							"revision": "dummyRevision",
							"relativePath": "dummyRelativePath"
						},
						"logging": {
							"elasticsearch": {
								%s
							}
						}
					}
				}`),
				tc.runIDJSON,
			)
			t.Log("input:", pipelineRunJSON)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx := mockServiceAccountTokenSecretRetriever(context.TODO(), mockCtrl)
			ctx, runInstance, cf := setupExaminee(ctx, t, pipelineRunJSON)
			ctx = withRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())

			// exercise
			err = runInstance.createTektonTaskRun(ctx)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, runInstance.pipelineRun)

			param := findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, param != nil)
			assert.Equal(t, tekton.ParamTypeString, param.Value.Type)
			assert.Equal(t, tc.expectedParamValue, param.Value.StringVal)

			param = findTaskRunParam(taskRun, TaskRunParamNameIndexURL)
			assert.Assert(t, is.Nil(param))
		})
	}

	/**
	 * Test: If there is no spec.logging.elasticsearch, the index URL
	 * template parameter should be defined as empty string, effectively
	 * disabling logging to Elasticsearch.
	 */
	test = "SuppressIndexURL"
	for _, tc := range []struct {
		name            string
		loggingFragment string
	}{
		{"NoLogging", ``},
		{"NoElasticsearch", `,"logging":{"___dummy___":1}`},
	} {
		t.Run(test+"_"+tc.name, func(t *testing.T) {
			var err error

			// setup
			pipelineRunJSON := fmt.Sprintf(fixIndent(`
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "PipelineRun",
					"metadata": {
						"name": "dummy1",
						"namespace": "namespace1"
					},
					"spec": {
						"jenkinsFile": {
							"repoUrl": "dummyRepoUrl",
							"revision": "dummyRevision",
							"relativePath": "dummyRelativePath"
						}
						%s
					}
				}`),
				tc.loggingFragment,
			)
			t.Log("input:", pipelineRunJSON)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx := mockServiceAccountTokenSecretRetriever(context.TODO(), mockCtrl)
			ctx, runInstance, cf := setupExaminee(ctx, t, pipelineRunJSON)

			// exercise
			err = runInstance.createTektonTaskRun(ctx)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, runInstance.pipelineRun)

			param := findTaskRunParam(taskRun, TaskRunParamNameIndexURL)
			assert.Assert(t, param != nil)
			assert.Equal(t, tekton.ParamTypeString, param.Value.Type)
			assert.Equal(t, "", param.Value.StringVal)

			param = findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, is.Nil(param))
		})
	}
}
