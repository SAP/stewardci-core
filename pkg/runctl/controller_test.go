package runctl

import (
	"context"
	"encoding/json"
	"errors"
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
	"gotest.tools/v3/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2/ktesting"
	"knative.dev/pkg/apis"

	_ "knative.dev/pkg/system/testing"
)

func Test_Controller_meterAllPipelineRunsPeriodic(t *testing.T) {
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

	c := NewController(
		ktesting.NewLogger(t, ktesting.DefaultConfig),
		cf,
		ControllerOpts{},
	)

	pipelineRun := fake.PipelineRun("r1", "ns1", api.PipelineSpec{})
	c.pipelineRunStore.Add(pipelineRun)

	deletedRun := fake.PipelineRun("r2", "ns1", api.PipelineSpec{})
	now := metav1.Now()
	deletedRun.SetDeletionTimestamp(&now)
	c.pipelineRunStore.Add(deletedRun)

	// VERIFY
	mockMetric.EXPECT().Observe(pipelineRun).Times(1)

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

func Test_Controller_syncHandler_PipelineRunNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	cf := newFakeClientFactory()
	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	mockPipelineRunFetcher.EXPECT().
		ByKey(ctx, gomock.Any()).
		Return(nil, nil)

	examinee := NewController(
		ktesting.NewLogger(t, ktesting.DefaultConfig),
		cf,
		ControllerOpts{},
	)

	examinee.pipelineRunFetcher = mockPipelineRunFetcher

	// EXERCISE
	err := examinee.syncHandler("foo/bar")

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

	controller := NewController(
		ktesting.NewLogger(t, ktesting.DefaultConfig),
		cf,
		ControllerOpts{},
	)

	controller.pipelineRunFetcher = k8s.NewClientBasedPipelineRunFetcher(client)
	controller.eventRecorder = record.NewFakeRecorder(20)
	return controller, cf
}

func getAPIPipelineRun(cf *fake.ClientFactory, name, namespace string) (*api.PipelineRun, error) {
	ctx := context.Background()
	cs := cf.StewardClientset()
	return cs.StewardV1alpha1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
}

func Test_Controller_syncHandler_deleted_unfinished(t *testing.T) {
	for _, currentState := range []api.State{
		api.StateUndefined,
		api.StateNew,
		api.StateWaiting,
		api.StatePreparing,
		api.StateRunning,
		api.StateCleaning,
	} {
		for _, test := range []struct {
			name                  string
			runManagerExpectation func(*runmocks.MockManager)
			withFinalizer         bool
			expectError           bool
			expectFinalizer       bool
			expectedState         api.State
			expectedResult        api.Result
		}{
			{
				name: "with_finalizer/cleanup_succeeds",
				runManagerExpectation: func(rm *runmocks.MockManager) {
					rm.EXPECT().
						Cleanup(gomock.Any(), gomock.Any()).
						Return(nil)
				},
				withFinalizer:   true,
				expectError:     false,
				expectFinalizer: false,
				expectedState:   api.StateFinished,
				expectedResult:  api.ResultDeleted,
			},
			{
				name: "with_finalizer/cleanup_fails",
				runManagerExpectation: func(rm *runmocks.MockManager) {
					rm.EXPECT().
						Cleanup(gomock.Any(), gomock.Any()).
						Return(errors.New("expected"))
				},
				withFinalizer:   true,
				expectError:     true,
				expectFinalizer: true,
				expectedState:   currentState,
				expectedResult:  api.ResultUndefined,
			},
			{
				name: "no_finalizer/cleanup_succeeds",
				runManagerExpectation: func(rm *runmocks.MockManager) {
					rm.EXPECT().
						Cleanup(gomock.Any(), gomock.Any()).
						Return(nil)
				},
				withFinalizer:   false,
				expectError:     false,
				expectFinalizer: false,
				expectedState:   api.StateFinished,
				expectedResult:  api.ResultDeleted,
			},
			{
				name: "no_finalizer/cleanup_fails",
				runManagerExpectation: func(rm *runmocks.MockManager) {
					rm.EXPECT().
						Cleanup(gomock.Any(), gomock.Any()).
						Return(errors.New("expected"))
				},
				withFinalizer:   false,
				expectError:     true,
				expectFinalizer: false,
				expectedState:   currentState,
				expectedResult:  api.ResultUndefined,
			},
		} {
			t.Run(fmt.Sprintf("state_%s/%s", currentState, test.name), func(t *testing.T) {
				// SETUP
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				pipelineRun := fake.PipelineRun("foo", "ns1", api.PipelineSpec{})
				now := metav1.Now()
				pipelineRun.SetDeletionTimestamp(&now)
				pipelineRun.Status.State = currentState

				if test.withFinalizer {
					pipelineRun.ObjectMeta.Finalizers = []string{k8s.FinalizerName}
				}

				controller, cf := newController(t, pipelineRun)
				runManager := runmocks.NewMockManager(mockCtrl)
				test.runManagerExpectation(runManager)

				controller.testing = &controllerTesting{
					createRunManagerStub:       newSimpleCreateRunManagerStub(runManager),
					loadPipelineRunsConfigStub: newEmptyRunsConfig,
				}

				// EXERCISE
				resultErr := controller.syncHandler("ns1/foo")

				// VERIFY
				if test.expectError {
					assert.Assert(t, resultErr != nil)
				} else {
					assert.NilError(t, resultErr)
				}
				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)
				t.Logf("Pipeline run result: status: %+v", result.Status)

				assert.Equal(t, test.expectedResult, result.Status.Result)
				assert.Equal(t, test.expectedState, result.Status.State)
				if test.expectFinalizer {
					assert.Assert(t, cmp.Contains(result.GetFinalizers(), k8s.FinalizerName))
				} else {
					assert.Assert(t, len(result.GetFinalizers()) == 0)
				}
			})
		}
	}
}

func Test_Controller_syncHandler_deleted_finished(t *testing.T) {
	t.Parallel()

	for _, currentResult := range []api.Result{
		api.ResultSuccess,
		api.ResultErrorConfig,
		api.ResultErrorContent,
		api.ResultErrorInfra,
		api.ResultAborted,
		api.ResultDeleted,
	} {
		for _, withFinalizer := range []bool{true, false} {
			var namePartFinalizer string
			if withFinalizer {
				namePartFinalizer = "with_finalizer"
			} else {
				namePartFinalizer = "no_finalizer"
			}

			t.Run(fmt.Sprintf("result_%s/%s", currentResult, namePartFinalizer), func(t *testing.T) {
				// SETUP
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				pipelineRun := fake.PipelineRun("foo", "ns1", api.PipelineSpec{})
				now := metav1.Now()
				pipelineRun.SetDeletionTimestamp(&now)
				pipelineRun.Status.State = api.StateFinished
				pipelineRun.Status.Result = currentResult

				if withFinalizer {
					pipelineRun.ObjectMeta.Finalizers = []string{k8s.FinalizerName}
				}

				controller, cf := newController(t, pipelineRun)
				runManager := runmocks.NewMockManager(mockCtrl)

				controller.testing = &controllerTesting{
					createRunManagerStub:       newSimpleCreateRunManagerStub(runManager),
					loadPipelineRunsConfigStub: newEmptyRunsConfig,
				}

				// EXERCISE
				resultErr := controller.syncHandler("ns1/foo")

				// VERIFY
				assert.NilError(t, resultErr)
				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)
				t.Logf("Pipeline run result: status: %+v", result.Status)

				assert.Equal(t, currentResult, result.Status.Result)
				assert.Equal(t, api.StateFinished, result.Status.State)
				assert.Assert(t, len(result.GetFinalizers()) == 0)
			})
		}
	}
}

func Test_Controller_syncHandler_new(t *testing.T) {
	error1 := errors.New("error1")
	errorRecoverable1 := serrors.Recoverable(errors.New("errorRecoverable1"))

	for _, currentState := range []api.State{
		api.StateUndefined,
		api.StateNew,
	} {
		for _, test := range []struct {
			name                       string
			pipelineRunSpec            api.PipelineSpec
			runManagerExpectation      func(*runmocks.MockManager, *runmocks.MockRun)
			loadPipelineRunsConfigStub func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error)
			isMaintenanceModeStub      func(ctx context.Context) (bool, error)
			expectedError              error
			expectedState              api.State
			expectedResult             api.Result
		}{
			{
				name:            "prepare_succeeds",
				pipelineRunSpec: api.PipelineSpec{},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Prepare(gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", "", nil)
				},
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name:            "prepare_fails",
				pipelineRunSpec: api.PipelineSpec{},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Prepare(gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", "", error1)
				},
				expectedError:  error1,
				expectedState:  api.StatePreparing,
				expectedResult: api.ResultUndefined,
			},
			{
				name:                  "maintenance_mode_check_fails",
				pipelineRunSpec:       api.PipelineSpec{},
				isMaintenanceModeStub: newIsMaintenanceModeStub(false, error1),
				expectedError:         error1,
				expectedState:         currentState,
				expectedResult:        api.ResultUndefined,
			},
			{
				name:                  "maintenance_mode_check_fails_but_returns_true",
				pipelineRunSpec:       api.PipelineSpec{},
				isMaintenanceModeStub: newIsMaintenanceModeStub(true, error1),
				expectedError:         error1,
				expectedState:         currentState,
				expectedResult:        api.ResultUndefined,
			},
			{
				name:                  "maintenance_mode",
				pipelineRunSpec:       api.PipelineSpec{},
				isMaintenanceModeStub: newIsMaintenanceModeStub(true, nil),
				expectedError:         errors.New("pipeline execution is paused while the system is in maintenance mode"),
				expectedState:         currentState,
				expectedResult:        api.ResultUndefined,
			},
			{
				name:            "get_pipelineruns_config_fails/unrecoverable",
				pipelineRunSpec: api.PipelineSpec{},
				loadPipelineRunsConfigStub: func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
					return nil, error1
				},
				expectedState:  api.StateFinished,
				expectedResult: api.ResultErrorInfra,
			},
			{
				name:            "get_pipelineruns_config_fails/recoverable",
				pipelineRunSpec: api.PipelineSpec{},
				loadPipelineRunsConfigStub: func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
					return nil, errorRecoverable1
				},
				expectedError:  errorRecoverable1,
				expectedState:  api.StatePreparing,
				expectedResult: api.ResultUndefined,
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				// SETUP
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				pipelineRun := fake.PipelineRun("foo", "ns1", test.pipelineRunSpec)
				pipelineRun.Status.State = currentState

				controller, cf := newController(t, pipelineRun)
				runManager := runmocks.NewMockManager(mockCtrl)
				runmock := runmocks.NewMockRun(mockCtrl)

				if test.runManagerExpectation != nil {
					test.runManagerExpectation(runManager, runmock)
				}
				if test.loadPipelineRunsConfigStub == nil {
					test.loadPipelineRunsConfigStub = newEmptyRunsConfig
				}
				if test.isMaintenanceModeStub == nil {
					test.isMaintenanceModeStub = newIsMaintenanceModeStub(false, nil)
				}

				controller.testing = &controllerTesting{
					createRunManagerStub:       newSimpleCreateRunManagerStub(runManager),
					loadPipelineRunsConfigStub: test.loadPipelineRunsConfigStub,
					isMaintenanceModeStub:      test.isMaintenanceModeStub,
				}

				// EXERCISE
				resultErr := controller.syncHandler("ns1/foo")

				// VERIFY
				if test.expectedError != nil {
					assert.Error(t, resultErr, test.expectedError.Error())
				} else {
					assert.NilError(t, resultErr)
				}

				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)

				t.Logf("Pipeline run result: status: %+v", result.Status)
				assert.Equal(t, test.expectedResult, result.Status.Result)
				assert.Equal(t, test.expectedState, result.Status.State)

				if test.expectedState == api.StateFinished {
					assert.Assert(t, len(result.ObjectMeta.Finalizers) == 0)
				} else {
					assert.Assert(t, cmp.Contains(result.GetFinalizers(), k8s.FinalizerName))
				}
			})
		}
	}
}

func Test_Controller_syncHandler_unfinished(t *testing.T) {
	error1 := errors.New("error1")
	errorRecoverable1 := serrors.Recoverable(errors.New("errorRecoverable1"))
	longAgo := metav1.Unix(10, 10)

	for _, maintenanceMode := range []bool{true, false} {

		for _, test := range []struct {
			name                       string
			pipelineRunSpec            api.PipelineSpec
			pipelineRunStatus          api.PipelineStatus
			startedAt                  metav1.Time
			runManagerExpectation      func(*runmocks.MockManager, *runmocks.MockRun)
			loadPipelineRunsConfigStub func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error)
			expectedError              error
			expectedState              api.State
			expectedResult             api.Result
			expectedMessage            string
		}{
			//----------------
			// preparing
			//----------------
			{
				name: "preparing/success",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Prepare(gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", "", nil)
				},
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "preparing/prepare_fails/error_unclassified",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Prepare(gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", "", error1)
				},
				expectedError:  error1,
				expectedState:  api.StatePreparing,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "preparing/prepare_fails/error_config",

				pipelineRunSpec: api.PipelineSpec{
					Secrets: []string{"secret1"},
				},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Prepare(gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", "", serrors.Classify(error1, api.ResultErrorContent))
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorContent,
				expectedMessage: "preparing failed: error1",
			},
			{
				name: "preparing/prepare_fails/error_content",

				pipelineRunSpec: api.PipelineSpec{
					Secrets: []string{"secret1"},
				},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Prepare(gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", "", serrors.Classify(error1, api.ResultErrorContent))
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorContent,
				expectedMessage: "preparing failed: error1",
			},
			{
				name: "preparing/prepare_fails/error_infra",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StatePreparing,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Prepare(gomock.Any(), gomock.Any(), gomock.Any()).
						Return("", "", serrors.Classify(error1, api.ResultErrorInfra))
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorInfra,
				expectedMessage: "preparing failed: error1",
			},
			//----------------
			// waiting
			//----------------
			{
				name: "waiting/taskrun_get_fails/unrecoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, error1)
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorInfra,
				expectedMessage: "waiting failed: error1",
			},
			{
				name: "waiting/taskrun_get_fails/recoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, errorRecoverable1)
				},
				expectedError:  errorRecoverable1,
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_deletion_pending/time_left",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)

					run.EXPECT().
						IsDeleted().
						Return(true).
						AnyTimes()
				},
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_deletion_pending/timeout",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				startedAt: longAgo,
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)

					run.EXPECT().
						IsDeleted().
						Return(true).
						AnyTimes()
				},
				expectedState:  api.StateCleaning,
				expectedResult: api.ResultErrorInfra,
			},
			{
				name: "waiting/get_pipelineruns_config_fails/unrecoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, nil)
				},
				loadPipelineRunsConfigStub: func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
					return nil, error1
				},
				expectedState:  api.StateCleaning,
				expectedResult: api.ResultErrorInfra,
			},
			{
				name: "waiting/get_pipelineruns_config_fails/recoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, nil)
				},
				loadPipelineRunsConfigStub: func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
					return nil, errorRecoverable1
				},
				expectedError:  errorRecoverable1,
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_start_succeeds",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, nil)
					rm.EXPECT().
						Start(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil)
				},
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_start_fails/unclassified",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, nil)
					rm.EXPECT().
						Start(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(error1)
				},
				expectedError:  error1,
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_start_fails/classified",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, nil)
					rm.EXPECT().
						Start(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(serrors.Classify(error1, api.ResultErrorConfig))
				},
				expectedState:  api.StateCleaning,
				expectedResult: api.ResultErrorConfig,
			},
			{
				name: "waiting/taskrun_neither_started_nor_finished/time_left",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)
					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					run.EXPECT().
						GetStartTime().
						Return(nil).
						AnyTimes()
					run.EXPECT().
						IsFinished().
						Return(false, api.ResultUndefined).
						AnyTimes()
				},
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_neither_started_nor_finished/timeout",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				startedAt: longAgo,
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)
					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					run.EXPECT().
						GetStartTime().
						Return(nil).
						MinTimes(1)
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorInfra,
				expectedMessage: `ERROR: waiting failed: main pod has not started after [0-9]+m[0-9]+s`,
			},
			{
				name: "waiting/taskrun_started_but_not_finished",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)
					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					now := metav1.Now()
					run.EXPECT().
						GetStartTime().
						Return(&now).
						AnyTimes()
					run.EXPECT().
						IsFinished().
						Return(false, api.ResultUndefined).
						AnyTimes()
				},
				expectedState:  api.StateRunning,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_started_and_finished",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)
					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					now := metav1.Now()
					run.EXPECT().
						GetStartTime().
						Return(&now).
						AnyTimes()
					run.EXPECT().
						IsFinished().
						Return(true, api.ResultSuccess).
						AnyTimes()
				},
				expectedState:  api.StateRunning,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_not_started_but_finished/restartable/delete_succeeds",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)

					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					run.EXPECT().
						GetStartTime().
						Return(nil).
						AnyTimes()
					run.EXPECT().
						IsFinished().
						AnyTimes().
						Return(true, api.ResultErrorInfra)
					run.EXPECT().
						IsRestartable().
						Return(true)

					rm.EXPECT().
						DeleteRun(gomock.Any(), gomock.Any()).
						Return(nil)
				},
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_not_started_but_finished/restartable/delete_fails/recoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)

					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					run.EXPECT().
						GetStartTime().
						Return(nil).
						AnyTimes()
					run.EXPECT().
						IsFinished().
						AnyTimes().
						Return(true, api.ResultErrorInfra)
					run.EXPECT().
						IsRestartable().
						Return(true)

					rm.EXPECT().
						DeleteRun(gomock.Any(), gomock.Any()).
						Return(errorRecoverable1)
				},
				expectedError:  errorRecoverable1,
				expectedState:  api.StateWaiting,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "waiting/taskrun_not_started_but_finished/restartable/delete_fails/unrecoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)

					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					run.EXPECT().
						GetStartTime().
						Return(nil).
						AnyTimes()
					run.EXPECT().
						IsFinished().
						AnyTimes().
						Return(true, api.ResultErrorInfra)
					run.EXPECT().
						IsRestartable().
						Return(true)

					rm.EXPECT().
						DeleteRun(gomock.Any(), gomock.Any()).
						Return(error1)
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorInfra,
				expectedMessage: "waiting failed: could not restart stuck task run",
			},
			{
				name: "waiting/taskrun_not_started_but_finished/not_restartable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateWaiting,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)

					run.EXPECT().
						IsDeleted().
						Return(false).
						AnyTimes()
					run.EXPECT().
						GetStartTime().
						Return(nil).
						AnyTimes()
					run.EXPECT().
						IsFinished().
						AnyTimes().
						Return(true, api.ResultErrorInfra)
					run.EXPECT().
						IsRestartable().
						Return(false)
				},
				expectedState:  api.StateCleaning,
				expectedResult: api.ResultErrorInfra,
			},
			//----------------
			// running
			//----------------
			{
				name: "running/taskrun_unfinished",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)
					run.EXPECT().
						GetContainerInfo().
						Return(nil)
					run.EXPECT().
						IsFinished().
						Return(false, api.ResultUndefined)
				},
				expectedState:  api.StateRunning,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "running/get_taskrun_fails/recoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, errorRecoverable1)
				},
				expectedError:  errorRecoverable1,
				expectedState:  api.StateRunning,
				expectedResult: api.ResultUndefined,
			},
			{
				name: "running/get_taskrun_fails/unrecoverable",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, error1)
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorInfra,
				expectedMessage: "running failed: error1",
			},
			{
				name: "running/taskrun_not_found",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(nil, nil)
				},
				expectedState:   api.StateCleaning,
				expectedResult:  api.ResultErrorInfra,
				expectedMessage: "running failed: task run not found in namespace .*",
			},
			{
				name: "running/taskrun_finished/timeout",

				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					run.EXPECT().
						GetContainerInfo().
						Return(&corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						})
					now := metav1.Now()
					run.EXPECT().
						GetCompletionTime().
						Return(&now)
					run.EXPECT().
						IsFinished().
						Return(true, api.ResultTimeout)
					run.EXPECT().
						GetMessage()
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)
				},
				expectedState:  api.StateCleaning,
				expectedResult: api.ResultTimeout,
			},
			{
				name:            "running/finished_terminated",
				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					run.EXPECT().
						GetContainerInfo().
						Return(&corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Message: "message",
							},
						})
					now := metav1.Now()
					run.EXPECT().
						IsFinished().
						Return(true, api.ResultSuccess)
					run.EXPECT().
						GetCompletionTime().
						Return(&now)
					run.EXPECT().
						GetMessage()
					rm.EXPECT().
						GetRun(gomock.Any(), gomock.Any()).
						Return(run, nil)
				},
				expectedState:  api.StateCleaning,
				expectedResult: api.ResultSuccess,
			},
			{
				name: "running/aborted_running",
				pipelineRunSpec: api.PipelineSpec{
					Intent: api.IntentAbort,
				},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateRunning,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Cleanup(gomock.Any(), gomock.Any()).
						Return(nil)
				},
				expectedState:  api.StateFinished,
				expectedResult: api.ResultAborted,
			},
			//----------------
			// cleaning
			//----------------
			// TODO add tests for state cleaning
			//----------------
			// finished
			//----------------
			{
				name:            "finished/result_undefined",
				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State:  api.StateFinished,
					Result: api.ResultUndefined,
				},
				expectedState:  api.StateFinished,
				expectedResult: api.ResultUndefined,
			},
			{
				name:            "finished/result_success",
				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State:  api.StateFinished,
					Result: api.ResultSuccess,
				},
				expectedState:  api.StateFinished,
				expectedResult: api.ResultSuccess,
			},
			{
				name:            "finished/result_error_infra",
				pipelineRunSpec: api.PipelineSpec{},
				pipelineRunStatus: api.PipelineStatus{
					State:  api.StateFinished,
					Result: api.ResultErrorInfra,
				},
				expectedState:  api.StateFinished,
				expectedResult: api.ResultErrorInfra,
			},
			//----------------
			// TODO
			//----------------
			{
				// TODO move into separate top-level test for abort
				name: "new/aborted",
				pipelineRunSpec: api.PipelineSpec{
					Intent: api.IntentAbort,
				},
				pipelineRunStatus: api.PipelineStatus{
					State: api.StateUndefined,
				},
				runManagerExpectation: func(rm *runmocks.MockManager, run *runmocks.MockRun) {
					rm.EXPECT().
						Cleanup(gomock.Any(), gomock.Any()).
						Return(nil)
				},
				expectedState:  api.StateFinished,
				expectedResult: api.ResultAborted,
			},
		} {
			t.Run(fmt.Sprintf("%s/maintenance_mode_%t", test.name, maintenanceMode), func(t *testing.T) {
				// SETUP
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				pipelineRun := fake.PipelineRun("foo", "ns1", test.pipelineRunSpec)
				pipelineRun.Status = test.pipelineRunStatus
				if test.startedAt.IsZero() {
					test.startedAt = metav1.Now()
				}
				pipelineRun.Status.StateDetails = api.StateItem{
					StartedAt: test.startedAt,
				}

				controller, cf := newController(t, pipelineRun)
				runManager := runmocks.NewMockManager(mockCtrl)
				runmock := runmocks.NewMockRun(mockCtrl)

				if test.runManagerExpectation != nil {
					test.runManagerExpectation(runManager, runmock)
				}
				if test.loadPipelineRunsConfigStub == nil {
					test.loadPipelineRunsConfigStub = newEmptyRunsConfig
				}

				controller.testing = &controllerTesting{
					createRunManagerStub:       newSimpleCreateRunManagerStub(runManager),
					loadPipelineRunsConfigStub: test.loadPipelineRunsConfigStub,
					isMaintenanceModeStub:      newIsMaintenanceModeStub(maintenanceMode, nil),
				}

				// EXERCISE
				resultErr := controller.syncHandler("ns1/foo")

				// VERIFY
				if test.expectedError != nil {
					assert.Error(t, resultErr, test.expectedError.Error())
				} else {
					assert.NilError(t, resultErr)
				}

				result, err := getAPIPipelineRun(cf, "foo", "ns1")
				assert.NilError(t, err)

				t.Logf("Pipeline run result: status: %+v", result.Status)
				assert.Equal(t, test.expectedResult, result.Status.Result)
				assert.Equal(t, test.expectedState, result.Status.State)

				if test.expectedMessage != "" {
					assert.Assert(t, cmp.Regexp(test.expectedMessage, result.Status.Message))
				}

				if test.expectedState == api.StateFinished {
					assert.Assert(t, len(result.ObjectMeta.Finalizers) == 0)
				} else {
					assert.Assert(t, cmp.Contains(result.GetFinalizers(), k8s.FinalizerName))
				}
			})
		}
	}
}

func Test_Controller_syncHandler_PipelineRunFetchFails_InternalServerError(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	cf := newFakeClientFactory()

	mockPipelineRunFetcher := mocks.NewMockPipelineRunFetcher(mockCtrl)
	message := "internal server error 1"
	mockPipelineRunFetcher.EXPECT().
		ByKey(ctx, gomock.Any()).
		Return(nil, k8serrors.NewInternalError(errors.New(message)))

	examinee := NewController(
		ktesting.NewLogger(t, ktesting.DefaultConfig),
		cf,
		ControllerOpts{},
	)
	examinee.pipelineRunFetcher = mockPipelineRunFetcher

	// EXERCISE
	err := examinee.syncHandler("foo/bar")

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
	// TODO avoid race condition with controller
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

	controller := NewController(
		ktesting.NewLogger(t, ktesting.DefaultConfig),
		cf,
		ControllerOpts{},
	)
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
	t.Log("Trigger controller stop")
	stopCh <- struct{}{}
}

func start(t *testing.T, controller *Controller, stopCh chan struct{}) {
	t.Helper()
	if err := controller.Run(1, stopCh); err != nil {
		t.Logf("Error running controller %s", err.Error())
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

func newSimpleCreateRunManagerStub(runManager run.Manager) func(k8s.PipelineRun) run.Manager {
	return func(k8s.PipelineRun) run.Manager {
		return runManager
	}
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
