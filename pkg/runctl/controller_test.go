package runctl

import (
	"fmt"
	"log"
	"strings"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	metrics "github.com/SAP/stewardci-core/pkg/metrics"
	run "github.com/SAP/stewardci-core/pkg/run"
	runmocks "github.com/SAP/stewardci-core/pkg/run/mocks"
	gomock "github.com/golang/mock/gomock"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	assert "gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

func Test_Controller_Success(t *testing.T) {
	t.Parallel()
	// SETUP
	cf := fake.NewClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		fake.ClusterRole(string(runClusterRoleName)),
	)
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(pr, cf)
	// VERIFY
	run, err := getPipelineRun("run1", "ns1", cf)
	assert.NilError(t, err)
	status := run.GetStatus()

	assert.Assert(t, !strings.Contains(status.Message, "ERROR"), status.Message)
	assert.Equal(t, api.StateWaiting, status.State)
	assert.Equal(t, 2, len(status.StateHistory))
}

func Test_Controller_Running(t *testing.T) {
	t.Parallel()
	// SETUP
	cf := fake.NewClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		fake.ClusterRole(string(runClusterRoleName)),
	)
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(pr, cf)
	// VERIFY
	run, err := getPipelineRun("run1", "ns1", cf)
	assert.NilError(t, err)
	runNs := run.GetRunNamespace()
	taskRun, _ := getTektonTaskRun(runNs, cf)
	now := metav1.Now()
	taskRun.Status.StartTime = &now
	updateTektonTaskRun(taskRun, runNs, cf)
	cf.Sleep("Waiting for Tekton TaskRun being started")
	run, err = getPipelineRun("run1", "ns1", cf)
	assert.NilError(t, err)
	status := run.GetStatus()
	assert.Equal(t, api.StateRunning, status.State)
}

func Test_Controller_Deletion(t *testing.T) {
	t.Parallel()
	// SETUP
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})
	cf := fake.NewClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		fake.ClusterRole(string(runClusterRoleName)),
	)

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(pr, cf)
	// VERIFY
	run, _ := getRun("run1", "ns1", cf)

	assert.Equal(t, 1, len(run.GetFinalizers()))

	now := metav1.Now()
	run.SetDeletionTimestamp(&now)
	updateRun(run, "ns1", cf)

	cf.Sleep("Wait for deletion")
	run, _ = getRun("run1", "ns1", cf)
	assert.Equal(t, 0, len(run.GetFinalizers()))

}

func Test_Controller_syncHandler_givesUp_onPipelineRunNotFound(t *testing.T) {
	t.Parallel()
	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := fake.NewClientFactory()
	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	mockPipelineRunFetcher.EXPECT().
		ByKey(gomock.Any()).
		Return(nil, nil)
	examinee := NewController(cf, metrics.NewMetrics())
	examinee.pipelineRunFetcher = mockPipelineRunFetcher

	// EXERCISE
	err := examinee.syncHandler("foo/bar")
	// VERIFY
	assert.NilError(t, err)
}

func newController(runs ...*api.PipelineRun) (*Controller, *fake.ClientFactory) {
	cf := fake.NewClientFactory(fake.ClusterRole(string(runClusterRoleName)))
	cs := cf.StewardClientset()
	cs.PrependReactor("create", "*", fake.NewCreationTimestampReactor())
	client := cf.StewardV1alpha1()
	for _, run := range runs {
		client.PipelineRuns(run.GetNamespace()).Create(run)
	}
	metrics := metrics.NewMetrics()
	controller := NewController(cf, metrics)
	controller.pipelineRunFetcher = k8s.NewClientBasedPipelineRunFetcher(client)
	return controller, cf
}

func getAPIPipelineRun(cf *fake.ClientFactory, name, namespace string) (*api.PipelineRun, error) {
	cs := cf.StewardClientset()
	return cs.StewardV1alpha1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})
}

func Test_Controller_syncHandler_delete(t *testing.T) {
	for _, test := range []struct {
		name                  string
		runManagerExpectation func(*runmocks.MockManager)
		hasFinalizer          bool
		expectedError         bool
		expectedFinalizer     bool
	}{

		{name: "delete with finalizer ok",
			runManagerExpectation: func(rm *runmocks.MockManager) {
				rm.EXPECT().Cleanup(gomock.Any()).Return(nil)
			},
			hasFinalizer:      true,
			expectedError:     false,
			expectedFinalizer: false,
		},
		{name: "delete without finalizer ok",
			runManagerExpectation: func(rm *runmocks.MockManager) {
				rm.EXPECT().Cleanup(gomock.Any()).Return(nil)
			},
			hasFinalizer:      false,
			expectedError:     false,
			expectedFinalizer: false,
		},
		{name: "delete with finalizer fail",
			runManagerExpectation: func(rm *runmocks.MockManager) {
				rm.EXPECT().Cleanup(gomock.Any()).Return(fmt.Errorf("expected"))
			},
			hasFinalizer:      true,
			expectedError:     true,
			expectedFinalizer: true,
		},
		{name: "delete without finalizer fail",
			runManagerExpectation: func(rm *runmocks.MockManager) {
				rm.EXPECT().Cleanup(gomock.Any()).Return(fmt.Errorf("expected"))
			},
			hasFinalizer:      false,
			expectedError:     true,
			expectedFinalizer: false, // TODO: is this the correct expect here?

		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()
			// SETUP
			run := fake.PipelineRun("foo", "ns1", api.PipelineSpec{})
			if test.hasFinalizer {
				run.ObjectMeta.Finalizers = []string{k8s.FinalizerName}
			}
			now := metav1.Now()
			run.SetDeletionTimestamp(&now)
			controller, cf := newController(run)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			runManager := runmocks.NewMockManager(mockCtrl)
			test.runManagerExpectation(runManager)
			controller.testing = &controllerTesting{runManagerStub: runManager}
			// EXERCISE
			err := controller.syncHandler("ns1/foo")
			// VERIFY
			if test.expectedError {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
			}
			result, err := getAPIPipelineRun(cf, "foo", "ns1")
			assert.NilError(t, err)
			log.Printf("%+v", result.Status)

			if test.expectedFinalizer {
				assert.Assert(t, len(result.GetFinalizers()) == 1)
			} else {
				assert.Assert(t, len(result.GetFinalizers()) == 0)
			}
		})
	}
}

func Test_Controller_syncHandler_mock(t *testing.T) {
	error1 := fmt.Errorf("error1")
	errorRecover1 := NewRecoverabilityInfoError(error1, true)
	for _, test := range []struct {
		name                  string
		pipelineSpec          api.PipelineSpec
		currentStatus         api.PipelineStatus
		runManagerExpectation func(*runmocks.MockManager, *runmocks.MockRun)
		expectedResult        api.Result
		expectedState         api.State
		expectedMessage       string
		expectedError         error
	}{
		{name: "new_ok",
			pipelineSpec:  api.PipelineSpec{},
			currentStatus: api.PipelineStatus{},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().Start(gomock.Any()).Return(nil)
			},
			expectedResult: api.ResultUndefined,
			expectedState:  api.StateWaiting,
		},
		{name: "preparing_ok",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StatePreparing,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().Start(gomock.Any()).Return(nil)
			},
			expectedResult: api.ResultUndefined,
			expectedState:  api.StateWaiting,
		},
		{name: "preparing_fail",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StatePreparing,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().Start(gomock.Any()).Return(error1)
			},
			expectedResult:  api.ResultUndefined,
			expectedState:   api.StatePreparing,
			expectedMessage: "error syncing resource .*error1",
			expectedError:   error1,
		},
		{name: "preparing_fail_content_error",
			pipelineSpec: api.PipelineSpec{
				Secrets: []string{"secret1"},
			},
			currentStatus: api.PipelineStatus{
				State: api.StatePreparing,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {

				rm.EXPECT().Start(gomock.Any()).Do(func(run k8s.PipelineRun) {
					run.UpdateResult(api.ResultErrorContent)
				}).Return(error1)
			},
			expectedResult:  api.ResultErrorContent,
			expectedState:   api.StateCleaning,
			expectedMessage: "error syncing resource .*error1",
		},
		{name: "waiting_fail",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateWaiting,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().GetRun(gomock.Any()).Return(nil, error1)
			},
			expectedResult: api.ResultErrorInfra,
			expectedState:  api.StateCleaning,
		},
		{name: "waiting_recover",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateWaiting,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().GetRun(gomock.Any()).Return(nil, errorRecover1)
			},
			expectedResult: api.ResultUndefined,
			expectedState:  api.StateWaiting,
			expectedError:  errorRecover1,
		},
		{name: "waiting_not_started",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateWaiting,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				run.EXPECT().GetStartTime().Return(nil)
				rm.EXPECT().GetRun(gomock.Any()).Return(run, nil)
			},
			expectedResult: "",
			expectedState:  api.StateWaiting,
		},
		{name: "waiting_started",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateWaiting,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				now := metav1.Now()
				run.EXPECT().GetStartTime().Return(&now)
				rm.EXPECT().GetRun(gomock.Any()).Return(run, nil)
			},
			expectedResult: "",
			expectedState:  api.StateRunning,
		},
		{name: "running_not_finished",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateRunning,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				run.EXPECT().GetContainerInfo().Return(nil)
				run.EXPECT().IsFinished().Return(false, api.ResultUndefined)
				rm.EXPECT().GetRun(gomock.Any()).Return(run, nil)
			},
			expectedResult: "",
			expectedState:  api.StateRunning,
		},
		{name: "running_get_error",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateRunning,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().GetRun(gomock.Any()).Return(nil, error1)
			},
			expectedResult:  "",
			expectedState:   api.StateCleaning,
			expectedMessage: "error syncing resource .*error1",
		},
		{name: "running_finished_timeout",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateRunning,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				run.EXPECT().GetContainerInfo().Return(
					&corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					})
				run.EXPECT().IsFinished().Return(true, api.ResultTimeout)
				run.EXPECT().GetSucceededCondition().Return(nil)
				rm.EXPECT().GetRun(gomock.Any()).Return(run, nil)
			},
			expectedResult: api.ResultTimeout,
			expectedState:  api.StateCleaning,
		},
		{name: "running_finished_terminated",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateRunning,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				run.EXPECT().GetContainerInfo().Return(
					&corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Message: "message",
						},
					})
				run.EXPECT().IsFinished().Return(true, api.ResultSuccess)
				rm.EXPECT().GetRun(gomock.Any()).Return(run, nil)
			},
			expectedResult: api.ResultSuccess,
			expectedState:  api.StateCleaning,
		},
		{name: "skip_new",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateNew,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
			},
			expectedResult: "",
			expectedState:  api.StateNew,
		},
		{name: "skip_finished",
			pipelineSpec: api.PipelineSpec{},
			currentStatus: api.PipelineStatus{
				State: api.StateFinished,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
			},
			expectedResult: "",
			expectedState:  api.StateFinished,
		},
		{name: "cleanup_abborted_new",
			pipelineSpec: api.PipelineSpec{
				Intent: api.IntentAbort,
			},
			currentStatus: api.PipelineStatus{
				State: api.StateUndefined,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().Cleanup(gomock.Any()).Return(nil)
			},
			expectedResult: api.ResultAborted,
			expectedState:  api.StateFinished,
		},
		{name: "cleanup_abborted_running",
			pipelineSpec: api.PipelineSpec{
				Intent: api.IntentAbort,
			},
			currentStatus: api.PipelineStatus{
				State: api.StateRunning,
			},
			runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				rm.EXPECT().Cleanup(gomock.Any()).Return(nil)
			},
			expectedResult: api.ResultAborted,
			expectedState:  api.StateFinished,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()
			// SETUP
			run := fake.PipelineRun("foo", "ns1", test.pipelineSpec)
			run.Status = test.currentStatus
			controller, cf := newController(run)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			runManager := runmocks.NewMockManager(mockCtrl)
			runmock := runmocks.NewMockRun(mockCtrl)
			test.runManagerExpectation(runManager, runmock)
			controller.testing = &controllerTesting{runManagerStub: runManager}
			// EXERCISE
			err := controller.syncHandler("ns1/foo")
			// VERIFY
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.NilError(t, err)
			}
			result, err := getAPIPipelineRun(cf, "foo", "ns1")
			assert.NilError(t, err)
			log.Printf("%+v", result.Status)
			assert.Equal(t, test.expectedResult, result.Status.Result, test.name)
			assert.Equal(t, test.expectedState, result.Status.State, test.name)

			if test.expectedMessage != "" {
				assert.Assert(t, is.Regexp(test.expectedMessage, result.Status.Message))
			}
		})
	}
}

func Test_Controller_syncHandler_initiatesRetrying_on500DuringPipelineRunFetch(t *testing.T) {
	t.Parallel()
	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := fake.NewClientFactory()
	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	message := "k8s kapot!"
	mockPipelineRunFetcher.EXPECT().
		ByKey(gomock.Any()).
		Return(nil, k8serrors.NewInternalError(fmt.Errorf(message)))

	examinee := NewController(cf, metrics.NewMetrics())
	examinee.pipelineRunFetcher = mockPipelineRunFetcher
	// EXERCISE
	err := examinee.syncHandler("foo/bar")
	// VERIFY
	assert.ErrorContains(t, err, message)
}

func Test_Controller_syncHandler_OnTimeout(t *testing.T) {
	t.Parallel()
	// SETUP
	cf := fake.NewClientFactory(

		// the tenant namespace
		fake.Namespace("tenant-ns-1"),

		// the Steward PipelineRun in status running
		StewardObjectFromJSON(t, `{
			"apiVersion": "steward.sap.com/v1alpha1",
			"kind": "PipelineRun",
			"metadata": {
				"name": "run1",
				"namespace": "tenant-ns-1",
				"uid": "a9e79ee8-69a8-4d8b-8a29-f51b53ada9b7"
			},
			"spec": {},
			"status": {
				"namespace": "steward-run-ns-1",
				"state": "running"
			}
		}`),

		// the run namespace
		// label is required for deletion
		CoreV1ObjectFromJSON(t, `{
			"apiVersion": "v1",
			"kind": "Namespace",
			"metadata": {
				"name": "steward-run-ns-1",
				"labels": {
					"id": "tenant1",
					"prefix": "steward-run"
				}
			}
		}`),

		// the Tekton TaskRun
		TektonObjectFromJSON(t, `{
			"apiVersion": "tekton.dev/v1alpha1",
			"kind": "TaskRun",
			"metadata": {
				"name": "steward-jenkinsfile-runner",
				"namespace": "steward-run-ns-1"
			},
			"spec": {},
			"status": {
				"conditions": [
					{
						"lastTransitionTime": "2019-09-16T12:55:40Z",
						"message": "message from Succeeded condition",
						"reason": "TaskRunTimeout",
						"status": "False",
						"type": "Succeeded"
					}
				],
				"startTime": "2019-09-16T12:45:40Z",
				"completionTime": "2019-09-16T12:55:40Z"
			}
		}`),

		fake.ClusterRole(string(runClusterRoleName)),
	)

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)

	// VERIFY
	run, err := getPipelineRun("run1", "tenant-ns-1", cf)
	assert.NilError(t, err)

	status := run.GetStatus()

	assert.Assert(t, status != nil)
	assert.Equal(t, api.StateFinished, status.State)
	assert.Equal(t, status.State, status.StateDetails.State)
	assert.Equal(t, api.ResultTimeout, status.Result)
	assert.Equal(t, "message from Succeeded condition", status.Message)
}

func newTestRunManager(workFactory k8s.ClientFactory, pipelineRunsConfig *pipelineRunsConfigStruct, secretProvider secrets.SecretProvider, namespaceManager k8s.NamespaceManager) run.Manager {
	runManager := NewRunManager(workFactory, pipelineRunsConfig, secretProvider, namespaceManager).(*runManager)
	runManager.testing = &runManagerTesting{
		getServiceAccountSecretNameStub: func(ctx *runContext) string { return "foo" },
	}
	return runManager
}

func startController(t *testing.T, cf *fake.ClientFactory) chan struct{} {
	cs := cf.StewardClientset()
	cs.PrependReactor("create", "*", fake.NewCreationTimestampReactor())
	stopCh := make(chan struct{}, 0)
	metrics := metrics.NewMetrics()
	controller := NewController(cf, metrics)
	controller.testing = &controllerTesting{newRunManagerStub: newTestRunManager}
	controller.pipelineRunFetcher = k8s.NewClientBasedPipelineRunFetcher(cf.StewardV1alpha1())
	cf.StewardInformerFactory().Start(stopCh)
	cf.TektonInformerFactory().Start(stopCh)
	go start(t, controller, stopCh)
	cf.Sleep("Wait for controller")
	return stopCh
}

func stopController(t *testing.T, stopCh chan struct{}) {
	log.Printf("Trigger controller stop")
	stopCh <- struct{}{}
}

func start(t *testing.T, controller *Controller, stopCh chan struct{}) {
	if err := controller.Run(1, stopCh); err != nil {
		t.Logf("Error running controller %s", err.Error())
	}
}

func resource(resource string) schema.GroupResource {
	return schema.GroupResource{Group: "", Resource: resource}
}

// GetPipelineRun returns the pipeline run with the given name in the given namespace.
// Return nil if not found.
func getPipelineRun(name string, namespace string, cf *fake.ClientFactory) (k8s.PipelineRun, error) {
	key := fake.ObjectKey(name, namespace)
	fetcher := k8s.NewClientBasedPipelineRunFetcher(cf.StewardV1alpha1())
	pipelineRun, err := fetcher.ByKey(key)
	if err != nil {
		return nil, err
	}
	return k8s.NewPipelineRun(pipelineRun, cf)
}

func createRun(run *api.PipelineRun, cf *fake.ClientFactory) error {
	_, err := cf.StewardV1alpha1().PipelineRuns(run.GetNamespace()).Create(run)
	if err == nil {
		cf.Sleep("wait for controller to pick up run")
	}
	return err
}

func getRun(name, namespace string, cf *fake.ClientFactory) (*api.PipelineRun, error) {
	return cf.StewardV1alpha1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})
}

func updateRun(run *api.PipelineRun, namespace string, cf *fake.ClientFactory) (*api.PipelineRun, error) {
	return cf.StewardV1alpha1().PipelineRuns(namespace).Update(run)
}

func getTektonTaskRun(namespace string, cf *fake.ClientFactory) (*tekton.TaskRun, error) {
	return cf.TektonV1alpha1().TaskRuns(namespace).Get(tektonTaskRunName, metav1.GetOptions{})
}

func updateTektonTaskRun(taskRun *tekton.TaskRun, namespace string, cf *fake.ClientFactory) (*tekton.TaskRun, error) {
	return cf.TektonV1alpha1().TaskRuns(namespace).Update(taskRun)
}
