package runctl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	cfg "github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/SAP/stewardci-core/pkg/runctl/constants"
	metricstesting "github.com/SAP/stewardci-core/pkg/runctl/metrics/testing"
	run "github.com/SAP/stewardci-core/pkg/runctl/run"
	runmocks "github.com/SAP/stewardci-core/pkg/runctl/run/mocks"
	"github.com/SAP/stewardci-core/pkg/runctl/runmgr"
	runctltesting "github.com/SAP/stewardci-core/pkg/runctl/testing"
	gomock "github.com/golang/mock/gomock"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	assert "gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	klog "k8s.io/klog/v2"
	"k8s.io/klog/v2/ktesting"
	"knative.dev/pkg/apis"
)

func Test_meterAllPipelineRunsPeriodic(t *testing.T) {
	// no parallel: patching global state

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetric := metricstesting.NewMockPipelineRunsMetric(mockCtrl)
	defer metricstesting.PatchPipelineRunsPeriodic(mockMetric)()
	cf := newFakeClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		runctltesting.FakeClusterRole(),
	)

	c := NewController(context.Background(), cf, ControllerOpts{})

	run := fake.PipelineRun("r1", "ns1", api.PipelineSpec{})
	c.pipelineRunStore.Add(run)

	deletedRun := fake.PipelineRun("r2", "ns1", api.PipelineSpec{})
	now := metav1.Now()
	deletedRun.SetDeletionTimestamp(&now)
	c.pipelineRunStore.Add(deletedRun)

	// VERIFY
	mockMetric.EXPECT().Observe(run).Times(1)

	// EXERCISE
	c.meterAllPipelineRunsPeriodic()
}

func Test_Controller_Success(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := newFakeClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		runctltesting.FakeClusterRole(),
	)

	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(t, pr, cf)

	// VERIFY
	run := getPipelineRun(t, "run1", "ns1", cf)
	status := run.GetStatus()

	assert.Assert(t, !strings.Contains(status.Message, "ERROR"), status.Message)
	assert.Equal(t, api.StateWaiting, status.State)
	assert.Equal(t, 2, len(status.StateHistory))
}

func Test_Controller_Running(t *testing.T) {
	t.Parallel()

	for _, containerState := range []string{
		"running",
		"terminated",
	} {
		t.Run(containerState, func(t *testing.T) {

			// SETUP
			cf := newFakeClientFactory(
				fake.SecretOpaque("secret1", "ns1"),
				runctltesting.FakeClusterRole(),
			)

			pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
				Secrets: []string{"secret1"},
			})

			// EXERCISE
			stopCh := startController(t, cf)
			defer stopController(t, stopCh)
			createRun(t, pr, cf)

			// VERIFY
			run := getPipelineRun(t, "run1", "ns1", cf)
			runNs := run.GetRunNamespace()
			taskRun := getTektonTaskRun(t, runNs, cf)
			now := metav1.Now()
			taskRun.Status.Steps = stepsWithContainer(containerState, now)
			condition := apis.Condition{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionUnknown,
				Reason: tekton.TaskRunReasonRunning.String(),
			}
			taskRun.Status.SetCondition(&condition)
			updateTektonTaskRun(t, taskRun, runNs, cf)
			cf.Sleep("Waiting for Tekton TaskRun being started")
			run = getPipelineRun(t, "run1", "ns1", cf)
			status := run.GetStatus()
			assert.Equal(t, api.StateRunning, status.State)
		})
	}
}

func stepsWithContainer(state string, startTime metav1.Time) []tekton.StepState {
	var stepState tekton.StepState
	time, _ := json.Marshal(startTime)
	s := fmt.Sprintf(`{ %q: {"startedAt": %s}, "container": %q, "name": "foo"}`, state, time, constants.JFRStepName)
	json.Unmarshal([]byte(s), &stepState)
	return []tekton.StepState{
		stepState,
	}
}

func Test_Controller_Deletion(t *testing.T) {
	t.Parallel()

	// SETUP
	pr := fake.PipelineRun("run1", "ns1", api.PipelineSpec{
		Secrets: []string{"secret1"},
	})
	cf := newFakeClientFactory(
		fake.SecretOpaque("secret1", "ns1"),
		runctltesting.FakeClusterRole(),
	)

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)
	createRun(t, pr, cf)

	// VERIFY
	run := getRun(t, "run1", "ns1", cf)

	assert.Equal(t, 1, len(run.GetFinalizers()))

	now := metav1.Now()
	run.SetDeletionTimestamp(&now)
	updateRun(t, run, "ns1", cf)

	cf.Sleep("Wait for deletion")
	run = getRun(t, "run1", "ns1", cf)
	assert.Equal(t, 0, len(run.GetFinalizers()))
}

func Test_Controller_syncHandler_givesUp_onPipelineRunNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := newFakeClientFactory()
	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	mockPipelineRunFetcher.EXPECT().
		ByKey(ctx, gomock.Any()).
		Return(nil, nil)
	examinee := NewController(context.Background(), cf, ControllerOpts{})
	examinee.pipelineRunFetcher = mockPipelineRunFetcher

	// EXERCISE
	err := examinee.syncHandler(
		klog.LoggerWithName(examinee.logger, "reconciler"),
		"foo/bar",
	)

	// VERIFY
	assert.NilError(t, err)
}

func newController(t *testing.T, runs ...*api.PipelineRun) (*Controller, *fake.ClientFactory) {
	t.Helper()
	ctx := context.Background()
	cf := newFakeClientFactory(runctltesting.FakeClusterRole())
	client := cf.StewardV1alpha1()
	for _, run := range runs {
		client.PipelineRuns(run.GetNamespace()).Create(ctx, run, metav1.CreateOptions{})
	}
	controller := NewController(context.Background(), cf, ControllerOpts{})
	controller.pipelineRunFetcher = k8s.NewClientBasedPipelineRunFetcher(client)
	controller.eventRecorder = record.NewFakeRecorder(20)
	return controller, cf
}

func getAPIPipelineRun(cf *fake.ClientFactory, name, namespace string) (*api.PipelineRun, error) {
	ctx := context.Background()
	cs := cf.StewardClientset()
	return cs.StewardV1alpha1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
}

func Test_Controller_syncHandler_delete(t *testing.T) {
	for _, currentState := range []api.State{
		api.StateNew,
		api.StateWaiting,
		api.StatePreparing,
		api.StateRunning,
		api.StateCleaning,
		api.StateUndefined,
	} {

		expectedStateOnError := currentState

		for _, test := range []struct {
			name                  string
			runManagerExpectation func(*runmocks.MockManager)
			hasFinalizer          bool
			expectedError         bool
			expectedFinalizer     bool
			expectedResult        api.Result
			expectedState         api.State
		}{

			{name: "delete with finalizer ok",
				runManagerExpectation: func(rm *runmocks.MockManager) {
					rm.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Return(nil)
				},
				hasFinalizer:      true,
				expectedError:     false,
				expectedFinalizer: false,
				expectedResult:    api.ResultDeleted,
				expectedState:     api.StateFinished,
			},
			{name: "delete with finalizer fail",
				runManagerExpectation: func(rm *runmocks.MockManager) {
					rm.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Return(fmt.Errorf("expected"))
				},
				hasFinalizer:      true,
				expectedError:     true,
				expectedFinalizer: true,
				expectedResult:    api.ResultUndefined,
				expectedState:     expectedStateOnError,
			},
			{name: "delete without finalizer ensure finished state",
				runManagerExpectation: func(rm *runmocks.MockManager) {
					rm.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Return(nil)
				},
				hasFinalizer:      false,
				expectedError:     false,
				expectedFinalizer: false,
				expectedResult:    api.ResultDeleted,
				expectedState:     api.StateFinished,
			},
		} {
			t.Run(fmt.Sprintf("%s current state %s", test.name, currentState), func(t *testing.T) {
				currentState := currentState
				test := test
				logger := ktesting.NewLogger(t, ktesting.NewConfig())
				t.Parallel()

				// SETUP
				run := fake.PipelineRun("foo", "ns1", api.PipelineSpec{})
				if test.hasFinalizer {
					run.ObjectMeta.Finalizers = []string{k8s.FinalizerName}
				}
				run.Status.State = currentState
				now := metav1.Now()
				run.SetDeletionTimestamp(&now)
				controller, cf := newController(t, run)
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()
				runManager := runmocks.NewMockManager(mockCtrl)
				test.runManagerExpectation(runManager)
				controller.testing = &controllerTesting{
					createRunManagerStub:       runManager,
					loadPipelineRunsConfigStub: newEmptyRunsConfig,
				}
				// EXERCISE
				err := controller.syncHandler(
					klog.LoggerWithName(controller.logger, "reconciler"),
					"ns1/foo",
				)

				// VERIFY
				if test.expectedError {
					assert.Assert(t, err != nil)
				} else {
					assert.NilError(t, err)
				}
				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)
				logger.Info(
					"Pipeline run result",
					"status", fmt.Sprintf("%+v", result.Status),
				)

				assert.Equal(t, test.expectedResult, result.Status.Result)
				assert.Equal(t, test.expectedState, result.Status.State)
				if test.expectedFinalizer {
					assert.Assert(t, len(result.GetFinalizers()) == 1)
				} else {
					assert.Assert(t, len(result.GetFinalizers()) == 0)
				}
			})
		}
	}
}

func Test_Controller_syncHandler_delete_on_finished_keeps_result_unchanged(t *testing.T) {
	for _, currentResult := range []api.Result{
		api.ResultDeleted,
		api.ResultSuccess,
		api.ResultErrorContent,
		api.ResultAborted,
		api.ResultErrorConfig,
		api.ResultErrorInfra} {
		for _, hasFinalizer := range []bool{true, false} {
			t.Run(fmt.Sprintf("finalizer %t current result %s", hasFinalizer, currentResult), func(t *testing.T) {
				currentResult := currentResult
				hasFinalizer := hasFinalizer
				logger := ktesting.NewLogger(t, ktesting.NewConfig())
				t.Parallel()

				// SETUP
				run := fake.PipelineRun("foo", "ns1", api.PipelineSpec{})
				if hasFinalizer {
					run.ObjectMeta.Finalizers = []string{k8s.FinalizerName}
				}
				run.Status.State = api.StateFinished
				run.Status.Result = currentResult
				now := metav1.Now()
				run.SetDeletionTimestamp(&now)
				controller, cf := newController(t, run)
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()
				runManager := runmocks.NewMockManager(mockCtrl)
				controller.testing = &controllerTesting{
					createRunManagerStub:       runManager,
					loadPipelineRunsConfigStub: newEmptyRunsConfig,
				}
				// EXERCISE
				err := controller.syncHandler(
					klog.LoggerWithName(controller.logger, "reconciler"),
					"ns1/foo",
				)

				// VERIFY
				assert.NilError(t, err)
				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)
				logger.Info(
					"Pipeline run result",
					"status", fmt.Sprintf("%+v", result.Status),
				)

				assert.Equal(t, currentResult, result.Status.Result)
				assert.Equal(t, api.StateFinished, result.Status.State)
				assert.Assert(t, len(result.GetFinalizers()) == 0)
			})
		}
	}
}

func Test_Controller_syncHandler_mock_start(t *testing.T) {
	error1 := fmt.Errorf("error1")
	errorRecover1 := serrors.Recoverable(error1)

	for _, currentStatus := range []api.PipelineStatus{
		{},
		{
			State: api.StateNew,
		},
	} {
		for _, test := range []struct {
			name                   string
			pipelineSpec           api.PipelineSpec
			runManagerExpectation  func(*runmocks.MockManager, *runmocks.MockRun)
			pipelineRunsConfigStub func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error)
			isMaintenanceModeStub  func(ctx context.Context) (bool, error)
			expectedResult         api.Result
			expectedState          api.State
			expectedMessage        string
			expectedError          error
		}{
			{
				name:         "new_ok",
				pipelineSpec: api.PipelineSpec{},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any()).Return("", "", nil)
				},
				pipelineRunsConfigStub: newEmptyRunsConfig,
				isMaintenanceModeStub:  newIsMaintenanceModeStub(false, nil),
				expectedResult:         api.ResultUndefined,
				expectedState:          api.StateWaiting,
			},
			{
				name:                   "new_maintenance_error_a",
				pipelineSpec:           api.PipelineSpec{},
				runManagerExpectation:  func(rm *runmocks.MockManager, run *runmocks.MockRun) {},
				pipelineRunsConfigStub: newEmptyRunsConfig,
				isMaintenanceModeStub:  newIsMaintenanceModeStub(false, error1),
				expectedResult:         api.ResultUndefined,
				expectedState:          api.StateNew,
				expectedError:          error1,
			},
			{
				name:                   "new_maintenance_error_b",
				pipelineSpec:           api.PipelineSpec{},
				runManagerExpectation:  func(rm *runmocks.MockManager, run *runmocks.MockRun) {},
				pipelineRunsConfigStub: newEmptyRunsConfig,
				isMaintenanceModeStub:  newIsMaintenanceModeStub(true, error1),
				expectedResult:         api.ResultUndefined,
				expectedState:          api.StateNew,
				expectedError:          error1,
			},
			{
				name:         "new_maintenance",
				pipelineSpec: api.PipelineSpec{},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				},
				pipelineRunsConfigStub: newEmptyRunsConfig,
				isMaintenanceModeStub:  newIsMaintenanceModeStub(true, nil),
				expectedResult:         api.ResultUndefined,
				expectedState:          api.StateNew,
				expectedError:          fmt.Errorf("pipeline execution is paused while the system is in maintenance mode"),
			},
			{
				name:                  "new_get_cofig_fail_not_recoverable",
				pipelineSpec:          api.PipelineSpec{},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {},
				pipelineRunsConfigStub: func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
					return nil, error1
				},
				isMaintenanceModeStub: newIsMaintenanceModeStub(false, nil),
				expectedResult:        api.ResultErrorInfra,
				expectedState:         api.StateFinished,
			},
			{
				name:         "new_get_cofig_fail_recoverable",
				pipelineSpec: api.PipelineSpec{},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				},
				pipelineRunsConfigStub: func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
					return nil, errorRecover1
				},
				isMaintenanceModeStub: newIsMaintenanceModeStub(false, nil),
				expectedResult:        api.ResultUndefined,
				expectedState:         api.StatePreparing,
				expectedError:         errorRecover1,
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				test := test
				t.Parallel()

				// SETUP
				run := fake.PipelineRun("foo", "ns1", test.pipelineSpec)
				run.Status = currentStatus
				controller, cf := newController(t, run)
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()
				runManager := runmocks.NewMockManager(mockCtrl)
				runmock := runmocks.NewMockRun(mockCtrl)
				test.runManagerExpectation(runManager, runmock)
				controller.testing = &controllerTesting{
					createRunManagerStub:       runManager,
					loadPipelineRunsConfigStub: test.pipelineRunsConfigStub,
					isMaintenanceModeStub:      test.isMaintenanceModeStub,
				}

				// EXERCISE
				resultErr := controller.syncHandler(
					klog.LoggerWithName(controller.logger, "reconciler"),
					"ns1/foo",
				)

				// VERIFY
				if test.expectedError != nil {
					assert.Error(t, resultErr, test.expectedError.Error())
				} else {
					assert.NilError(t, resultErr)
				}

				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)
				t.Logf("%+v", result.Status)
				assert.Equal(t, test.expectedResult, result.Status.Result, test.name)
				assert.Equal(t, test.expectedState, result.Status.State, test.name)

				if test.expectedMessage != "" {
					assert.Assert(t, is.Regexp(test.expectedMessage, result.Status.Message))
				}

				if test.expectedState == api.StateFinished {
					assert.Assert(t, len(result.ObjectMeta.Finalizers) == 0)
				} else {
					assert.Assert(t, len(result.ObjectMeta.Finalizers) == 1)
				}
			})
		}
	}
}

func Test_Controller_syncHandler_mock(t *testing.T) {
	error1 := fmt.Errorf("error1")
	errorRecover1 := serrors.Recoverable(error1)
	longAgo := metav1.Unix(10, 10)

	for _, maintenanceMode := range []bool{true, false} {

		for _, test := range []struct {
			name                       string
			pipelineSpec               api.PipelineSpec
			currentStatus              api.PipelineStatus
			startedAt                  metav1.Time
			runManagerExpectation      func(*runmocks.MockManager, *runmocks.MockRun)
			loadPipelineRunsConfigStub func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error)
			expectedResult             api.Result
			expectedState              api.State
			expectedMessage            string
			expectedError              error
		}{
			{
				name:         "preparing_ok",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any()).Return("", "", nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultUndefined,
				expectedState:              api.StateWaiting,
			},
			{
				name:         "preparing_fail",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any()).Return("", "", error1)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultUndefined,
				expectedState:              api.StatePreparing,
				expectedMessage:            "",
				expectedError:              error1,
			},
			{
				name: "preparing_fail_on_content_error_during_start",
				pipelineSpec: api.PipelineSpec{
					Secrets: []string{"secret1"},
				},
				currentStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {

					rm.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any()).Return("", "", serrors.Classify(error1, api.ResultErrorContent))
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultErrorContent,
				expectedState:              api.StateCleaning,
				expectedMessage:            "preparing failed .*error1",
			},
			{
				name:         "waiting_fail",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(nil, error1)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultErrorInfra,
				expectedState:              api.StateCleaning,
			},
			{
				name:         "waiting_recover",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(nil, errorRecover1)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultUndefined,
				expectedState:              api.StateWaiting,
				expectedError:              errorRecover1,
			},
			{
				name:         "waiting_start_success",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(nil, nil)
					rm.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "",
				expectedState:              api.StateWaiting,
			},
			{
				name:         "waiting_start_fail",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(nil, nil)
					rm.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any()).Return(error1)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultUndefined,
				expectedState:              api.StateWaiting,
				expectedError:              error1,
			},
			{
				name:         "waiting_fail_on_load_pipeline_run_config",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {},
				loadPipelineRunsConfigStub: func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
					return nil, error1
				},
				expectedResult: api.ResultErrorInfra,
				expectedState:  api.StateCleaning,
			},
			{
				name:         "waiting_start_fail_on_config_error_during_start",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(nil, nil)
					rm.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any()).Return(serrors.Classify(error1, api.ResultErrorConfig))
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultErrorConfig,
				expectedState:              api.StateCleaning,
			},
			{
				name:         "waiting_not_started",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					run.EXPECT().GetStartTime().Return(nil)
					run.EXPECT().IsRestartable().Return(false)
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "",
				expectedState:              api.StateWaiting,
			},
			{
				name:         "waiting_started",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					now := metav1.Now()
					run.EXPECT().GetStartTime().Return(&now)
					run.EXPECT().IsRestartable().Return(false)
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "",
				expectedState:              api.StateRunning,
			},
			{
				name:         "waiting_timeout",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				startedAt: longAgo,
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultErrorInfra,
				expectedState:              api.StateCleaning,
				expectedMessage:            `ERROR: waiting failed \[PipelineRun{name: foo, namespace: ns1, state: waiting}\]: main pod has not started after 10m0s`,
			},

			{
				name:         "waiting_restartable",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					run.EXPECT().IsRestartable().Return(true)

					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
					rm.EXPECT().DeleteRun(gomock.Any(), gomock.Any()).Return(nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "",
				expectedState:              api.StateWaiting,
			},
			{
				name:         "waiting_restartable_delete_fails",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					run.EXPECT().IsRestartable().Return(true)

					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
					rm.EXPECT().DeleteRun(gomock.Any(), gomock.Any()).Return(error1)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultErrorInfra,
				expectedState:              api.StateCleaning,
				expectedMessage:            "restart failed",
			},
			{
				name:         "running_not_finished",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					run.EXPECT().GetContainerInfo().Return(nil)
					run.EXPECT().IsFinished().Return(false, api.ResultUndefined)
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "",
				expectedState:              api.StateRunning,
			},
			{
				name:         "running_recover",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, errorRecover1)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "",
				expectedState:              api.StateRunning,
				expectedError:              errorRecover1,
			},
			{
				name:         "running_get_error",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(nil, error1)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "error_infra",
				expectedState:              api.StateCleaning,
				expectedMessage:            "running failed .*error1",
			},
			{
				name:         "running_get_not_found",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(nil, nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "error_infra",
				expectedState:              api.StateCleaning,
				expectedMessage:            "running failed .* task run not found in namespace.*",
			},
			{
				name:         "running_finished_timeout",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					run.EXPECT().GetContainerInfo().Return(
						&corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						})
					now := metav1.Now()
					run.EXPECT().GetCompletionTime().Return(&now)
					run.EXPECT().IsFinished().Return(true, api.ResultTimeout)
					run.EXPECT().GetMessage()
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultTimeout,
				expectedState:              api.StateCleaning,
			},
			{
				name:         "running_finished_terminated",
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
					now := metav1.Now()
					run.EXPECT().IsFinished().Return(true, api.ResultSuccess)
					run.EXPECT().GetCompletionTime().Return(&now)
					run.EXPECT().GetMessage()
					rm.EXPECT().GetRun(gomock.Any(), gomock.Any()).Return(run, nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultSuccess,
				expectedState:              api.StateCleaning,
			},
			{
				name:         "skip_finished",
				pipelineSpec: api.PipelineSpec{},
				currentStatus: api.PipelineStatus{
					State: api.StateFinished,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             "",
				expectedState:              api.StateFinished,
			},
			{
				name: "cleanup_abborted_new",
				pipelineSpec: api.PipelineSpec{
					Intent: api.IntentAbort,
				},
				currentStatus: api.PipelineStatus{
					State: api.StateUndefined,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Return(nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultAborted,
				expectedState:              api.StateFinished,
			},
			{
				name: "cleanup_abborted_running",
				pipelineSpec: api.PipelineSpec{
					Intent: api.IntentAbort,
				},
				currentStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Return(nil)
				},
				loadPipelineRunsConfigStub: newEmptyRunsConfig,
				expectedResult:             api.ResultAborted,
				expectedState:              api.StateFinished,
			},
		} {
			t.Run(fmt.Sprintf("%+s_maintenanceMode_%t", test.name, maintenanceMode), func(t *testing.T) {
				test := test
				t.Parallel()

				// SETUP
				run := fake.PipelineRun("foo", "ns1", test.pipelineSpec)
				run.Status = test.currentStatus
				if test.startedAt.IsZero() {
					test.startedAt = metav1.Now()
				}
				run.Status.StateDetails = api.StateItem{
					StartedAt: test.startedAt,
				}
				controller, cf := newController(t, run)
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()
				runManager := runmocks.NewMockManager(mockCtrl)
				runmock := runmocks.NewMockRun(mockCtrl)
				test.runManagerExpectation(runManager, runmock)
				controller.testing = &controllerTesting{
					createRunManagerStub:       runManager,
					loadPipelineRunsConfigStub: test.loadPipelineRunsConfigStub,
					isMaintenanceModeStub:      newIsMaintenanceModeStub(maintenanceMode, nil),
				}

				// EXERCISE
				err := controller.syncHandler(
					klog.LoggerWithName(controller.logger, "reconciler"),
					"ns1/foo",
				)

				// VERIFY
				if test.expectedError != nil {
					assert.Equal(t, test.expectedError, err)
				} else {
					assert.NilError(t, err)
				}

				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)
				t.Logf("%+v", result.Status)
				assert.Equal(t, test.expectedResult, result.Status.Result, test.name)
				assert.Equal(t, test.expectedState, result.Status.State, test.name)

				if test.expectedMessage != "" {
					assert.Assert(t, is.Regexp(test.expectedMessage, result.Status.Message))
				}

				if test.expectedState == api.StateFinished {
					assert.Assert(t, len(result.ObjectMeta.Finalizers) == 0)
				} else {
					assert.Assert(t, len(result.ObjectMeta.Finalizers) == 1)
				}
			})
		}
	}
}

func Test_Controller_syncHandler_initiatesRetrying_on500DuringPipelineRunFetch(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cf := newFakeClientFactory()
	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	message := "k8s kapot!"
	mockPipelineRunFetcher.EXPECT().
		ByKey(ctx, gomock.Any()).
		Return(nil, k8serrors.NewInternalError(fmt.Errorf(message)))

	examinee := NewController(context.Background(), cf, ControllerOpts{})
	examinee.pipelineRunFetcher = mockPipelineRunFetcher

	// EXERCISE
	err := examinee.syncHandler(
		klog.LoggerWithName(examinee.logger, "reconciler"),
		"foo/bar",
	)

	// VERIFY
	assert.ErrorContains(t, err, message)
}

func Test_Controller_syncHandler_OnTimeout(t *testing.T) {
	t.Parallel()

	// SETUP
	cf := newFakeClientFactory(

		// the content namespace
		fake.Namespace("content-ns-1"),

		// the Steward PipelineRun in status running
		runctltesting.StewardObjectFromJSON(t, `{
			"apiVersion": "steward.sap.com/v1alpha1",
			"kind": "PipelineRun",
			"metadata": {
				"name": "run1",
				"namespace": "content-ns-1",
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
		runctltesting.CoreV1ObjectFromJSON(t, `{
			"apiVersion": "v1",
			"kind": "Namespace",
			"metadata": {
				"name": "steward-run-ns-1",
				"labels": {
					"id": "content1",
					"prefix": "steward-run"
				}
			}
		}`),

		// the Tekton TaskRun
		runctltesting.TektonObjectFromJSON(t, `{
			"apiVersion": "tekton.dev/v1beta1",
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
		runctltesting.FakeClusterRole(),
	)

	// EXERCISE
	stopCh := startController(t, cf)
	defer stopController(t, stopCh)

	// VERIFY
	run := getPipelineRun(t, "run1", "content-ns-1", cf)
	status := run.GetStatus()

	assert.Assert(t, status != nil)
	assert.Equal(t, api.StateFinished, status.State)
	assert.Equal(t, status.State, status.StateDetails.State)
	assert.Equal(t, api.ResultTimeout, status.Result)
	assert.Equal(t, "message from Succeeded condition", status.Message)
}

func newTestRunManager(workFactory k8s.ClientFactory, secretProvider secrets.SecretProvider) run.Manager {
	return runmgr.NewRunManager(workFactory, secretProvider)
}

func startController(t *testing.T, cf *fake.ClientFactory) chan struct{} {
	stopCh := make(chan struct{})
	controller := NewController(context.Background(), cf, ControllerOpts{})
	controller.testing = &controllerTesting{
		newRunManagerStub:          newTestRunManager,
		loadPipelineRunsConfigStub: newEmptyRunsConfig,
		isMaintenanceModeStub:      newIsMaintenanceModeStub(false, nil),
	}
	controller.pipelineRunFetcher = k8s.NewClientBasedPipelineRunFetcher(cf.StewardV1alpha1())

	cf.StewardInformerFactory().Start(stopCh)
	cf.TektonInformerFactory().Start(stopCh)
	go start(t, controller, stopCh)
	cf.Sleep("Wait for controller")
	return stopCh
}

func stopController(t *testing.T, stopCh chan struct{}) {
	logger := ktesting.NewLogger(t, ktesting.NewConfig())
	logger.Info("Trigger controller stop")
	stopCh <- struct{}{}
}

func start(t *testing.T, controller *Controller, stopCh chan struct{}) {
	t.Helper()
	logger := ktesting.NewLogger(t, ktesting.NewConfig())
	if err := controller.Run(1, stopCh); err != nil {
		logger.Error(err, "Error running controller")
	}
}

func getPipelineRun(t *testing.T, name string, namespace string, cf *fake.ClientFactory) k8s.PipelineRun {
	t.Helper()
	ctx := context.Background()
	key := fake.ObjectKey(name, namespace)
	fetcher := k8s.NewClientBasedPipelineRunFetcher(cf.StewardV1alpha1())
	pipelineRun, err := fetcher.ByKey(ctx, key)
	if err != nil {
		t.Fatalf("could not get pipeline run: %s", err.Error())
	}
	wrapper, err := k8s.NewPipelineRun(ctx, pipelineRun, cf)
	if err != nil {
		t.Fatalf("could not get pipeline run: %s", err.Error())
	}
	return wrapper
}

func createRun(t *testing.T, run *api.PipelineRun, cf *fake.ClientFactory) {
	t.Helper()
	ctx := context.Background()
	_, err := cf.StewardV1alpha1().PipelineRuns(run.GetNamespace()).Create(ctx, run, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create pipeline run: %s", err.Error())
	}
	cf.Sleep("wait for controller to pick up run")
}

func getRun(t *testing.T, name, namespace string, cf *fake.ClientFactory) *api.PipelineRun {
	t.Helper()
	ctx := context.Background()
	run, err := cf.StewardV1alpha1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("could not get run: %s", err.Error())
	}
	return run
}

func updateRun(t *testing.T, run *api.PipelineRun, namespace string, cf *fake.ClientFactory) *api.PipelineRun {
	t.Helper()
	ctx := context.Background()
	updated, err := cf.StewardV1alpha1().PipelineRuns(namespace).Update(ctx, run, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("could not update run: %s", err.Error())
	}
	return updated
}

func getTektonTaskRun(t *testing.T, namespace string, cf *fake.ClientFactory) *tekton.TaskRun {
	t.Helper()
	ctx := context.Background()
	taskRun, err := cf.TektonV1beta1().TaskRuns(namespace).Get(ctx, constants.TektonTaskRunName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("could not get Tekton task run: %s", err.Error())
	}
	return taskRun
}

func updateTektonTaskRun(t *testing.T, taskRun *tekton.TaskRun, namespace string, cf *fake.ClientFactory) *tekton.TaskRun {
	t.Helper()
	ctx := context.Background()
	updated, err := cf.TektonV1beta1().TaskRuns(namespace).Update(ctx, taskRun, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("could not update Tekton task run: %s", err.Error())
	}
	return updated
}

func newFakeClientFactory(objects ...runtime.Object) *fake.ClientFactory {
	cf := fake.NewClientFactory(objects...)

	cf.KubernetesClientset().PrependReactor("create", "*", fake.GenerateNameReactor(0))

	cf.StewardClientset().PrependReactor("create", "*", fake.NewCreationTimestampReactor())

	return cf
}

func newIsMaintenanceModeStub(maintenanceMode bool, err error) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		return maintenanceMode, err
	}
}

func newEmptyRunsConfig(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
	return &cfg.PipelineRunsConfigStruct{},
		nil
}
