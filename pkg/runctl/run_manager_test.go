package runctl

import (
	"fmt"
	"strings"
	"testing"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	fsteward "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/fake"
	"github.com/SAP/stewardci-core/pkg/k8s"
	k8sfake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	tektonclientfake "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/fake"
	"github.com/davecgh/go-spew/spew"
	gomock "github.com/golang/mock/gomock"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func Test_RunManager_PrepareRunNamespace_CreatesNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	preparePredefinedSecrets(mockSecretProvider)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)

	examinee := NewRunManager(mockFactory, mockSecretProvider, mockNamespaceManager).(*runManager)

	// EXERCISE
	err := examinee.prepareRunNamespace(mockPipelineRun)
	assert.NilError(t, err)

	// VERIFY
	assert.Assert(t, strings.HasPrefix(mockPipelineRun.GetRunNamespace(), runNamespacePrefix))
}

func Test_RunManager_Start_CreatesTektonTaskRun(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	preparePredefinedSecrets(mockSecretProvider)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)

	examinee := NewRunManager(mockFactory, mockSecretProvider, mockNamespaceManager)

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.NilError(t, err)

	// VERIFY
	result, err := mockFactory.TektonV1alpha1().TaskRuns(mockPipelineRun.GetRunNamespace()).Get(
		tektonTaskRunName, metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Assert(t, result != nil)
}

func Test_RunManager_Start_DoesNotSetPipelineRunStatus(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	preparePredefinedSecrets(mockSecretProvider)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)

	examinee := NewRunManager(mockFactory, mockSecretProvider, mockNamespaceManager)

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.NilError(t, err)

	// VERIFY
	// UpdateState should never be called by BuildStarter
	mockPipelineRun.EXPECT().UpdateState(gomock.Any()).Times(0)
}

func Test_RunManager_Start_DoesCopySecret(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "scm_secret1"
	spec := &steward.PipelineSpec{JenkinsFile: steward.JenkinsFile{Secret: secretName}}
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocksWithSpec(mockCtrl, spec)

	preparePredefinedSecrets(mockSecretProvider, secretName)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)

	examinee := NewRunManager(mockFactory, mockSecretProvider, mockNamespaceManager)

	// VERIFY
	// UpdateState should never be called by BuildStarter
	mockPipelineRun.EXPECT().UpdateState(gomock.Any()).Times(0)

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.NilError(t, err)

}

func Test_RunManager_Cleanup_RemovesNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	preparePredefinedSecrets(mockSecretProvider)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	mockPipelineRun.EXPECT().FinishState()

	examinee := NewRunManager(mockFactory, mockSecretProvider, mockNamespaceManager).(*runManager)
	err := examinee.prepareRunNamespace(mockPipelineRun)
	assert.NilError(t, err)
	//TODO: mockNamespaceManager.EXPECT().Create()...

	// EXERCISE
	examinee.Cleanup(mockPipelineRun)
	//TODO: mockNamespaceManager.EXPECT().Delete()...
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
		t *testing.T, pipelineRunJSON string,
	) (
		examinee *runManager, k8sPipelineRun k8s.PipelineRun, cf *k8sfake.ClientFactory,
	) {
		pipelineRun := StewardObjectFromJSON(t, pipelineRunJSON).(*steward.PipelineRun)
		t.Log("decoded:\n", spew.Sdump(pipelineRun))

		cf = k8sfake.NewClientFactory(
			k8sfake.Namespace("namespace1"),
			pipelineRun,
		)
		k8sPipelineRun, err := k8s.NewPipelineRunFetcher(cf).ByName("namespace1", "dummy1")
		assert.NilError(t, err)
		examinee = NewRunManager(
			cf,
			k8s.NewTenantNamespace(cf, pipelineRun.GetNamespace()).GetSecretProvider(),
			k8s.NewNamespaceManager(cf, "prefix1", 0),
		).(*runManager)
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
			examinee, k8sPipelineRun, cf := setupExaminee(t, pipelineRunJSON)

			// exercise
			err = examinee.createTektonTaskRun(k8sPipelineRun)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, k8sPipelineRun)

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
			examinee, k8sPipelineRun, cf := setupExaminee(t, pipelineRunJSON)

			// exercise
			err = examinee.createTektonTaskRun(k8sPipelineRun)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, k8sPipelineRun)

			param := findTaskRunParam(taskRun, TaskRunParamNameIndexURL)
			assert.Assert(t, param != nil)
			assert.Equal(t, tekton.ParamTypeString, param.Value.Type)
			assert.Equal(t, "", param.Value.StringVal)

			param = findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, is.Nil(param))
		})
	}
}

func preparePredefinedSecrets(mockSecretProvider *secretMocks.MockSecretProvider, secrets ...string) {
	for _, secret := range secrets {
		mockSecretProvider.EXPECT().GetSecret(secret).Return(&v1.Secret{}, nil)
	}
}

func preparePredefinedClusterRole(t *testing.T, factory *mocks.MockClientFactory, pipelineRun *mocks.MockPipelineRun) {
	// Uncomment this if unexpected call to FinishState() swallows the error you want to see
	// pipelineRun.EXPECT().FinishState().AnyTimes()

	// Create expected cluster role
	_, err := factory.RbacV1beta1().ClusterRoles().Create(k8sfake.ClusterRole(string(runClusterRoleName)))
	assert.NilError(t, err)
}

func prepareMocks(ctrl *gomock.Controller) (*mocks.MockClientFactory, *mocks.MockPipelineRun, *secretMocks.MockSecretProvider, k8s.NamespaceManager) {
	return prepareMocksWithSpec(ctrl, &steward.PipelineSpec{})
}

func prepareMocksWithSpec(ctrl *gomock.Controller, spec *steward.PipelineSpec) (*mocks.MockClientFactory, *mocks.MockPipelineRun, *secretMocks.MockSecretProvider, k8s.NamespaceManager) {
	mockFactory := mocks.NewMockClientFactory(ctrl)

	coreClientSet := kubefake.NewSimpleClientset()
	mockFactory.EXPECT().CoreV1().Return(coreClientSet.CoreV1()).AnyTimes()
	mockFactory.EXPECT().RbacV1beta1().Return(coreClientSet.RbacV1beta1()).AnyTimes()

	stewardClientset := fsteward.NewSimpleClientset()
	mockFactory.EXPECT().StewardV1alpha1().Return(stewardClientset.StewardV1alpha1()).AnyTimes()

	tektonClientset := tektonclientfake.NewSimpleClientset()
	mockFactory.EXPECT().TektonV1alpha1().Return(tektonClientset.TektonV1alpha1()).AnyTimes()

	runNamespace := ""
	mockPipelineRun := mocks.NewMockPipelineRun(ctrl)
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockPipelineRun.EXPECT().GetStatus().Return(&steward.PipelineStatus{}).AnyTimes()
	mockPipelineRun.EXPECT().GetKey().Return("key").AnyTimes()
	mockPipelineRun.EXPECT().GetRepoServerURL().Return("server", nil).AnyTimes()
	mockPipelineRun.EXPECT().GetRunNamespace().DoAndReturn(func() string {
		return runNamespace
	}).AnyTimes()

	mockPipelineRun.EXPECT().UpdateRunNamespace(gomock.Any()).Do(func(arg string) {
		runNamespace = arg
	})

	mockSecretProvider := secretMocks.NewMockSecretProvider(ctrl)

	//TODO: Mock when required
	namespaceManager := k8s.NewNamespaceManager(mockFactory, runNamespacePrefix, runNamespaceRandomLength)

	return mockFactory, mockPipelineRun, mockSecretProvider, namespaceManager
}
