package k8s

import (
	"errors"
	"fmt"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	is "gotest.tools/assert/cmp"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/retry"
)

const message string = "MyMessage"

func Test_pipelineRun_GetRunNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	run := &api.PipelineRun{
		Status: api.PipelineStatus{
			Namespace: "foo",
		},
	}
	examinee, err := NewPipelineRun(run, nil)
	assert.NilError(t, err)

	// EXERCISE
	ns := examinee.GetRunNamespace()

	// VERIFY
	assert.Equal(t, "foo", ns)
}

func Test_pipelineRun_GetKey(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee, err := NewPipelineRun(run, nil)
	assert.NilError(t, err)

	// EXERCISE
	key := examinee.GetKey()

	// VERIFY
	assert.Equal(t, "ns1/foo", key)
}

func Test_pipelineRun_GetNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee, err := NewPipelineRun(run, nil)
	assert.NilError(t, err)

	// EXERCISE
	key := examinee.GetNamespace()

	// VERIFY
	assert.Equal(t, "ns1", key)
}

func Test_pipelineRun_GetName(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee, err := NewPipelineRun(run, nil)
	assert.NilError(t, err)

	// EXERCISE
	key := examinee.GetName()

	// VERIFY
	assert.Equal(t, "foo", key)
}

func Test_NewPipelineRun_IsCopy(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	run.Status.Result = api.ResultUndefined
	factory := fake.NewClientFactory(run)

	// EXERCISE
	examinee, err := NewPipelineRun(run, factory)

	// VERIFY
	assert.NilError(t, err)
	examinee.UpdateResult(api.ResultSuccess, metav1.Now())
	assert.Equal(t, api.ResultUndefined, run.Status.Result)
	assert.Equal(t, api.ResultSuccess, examinee.GetStatus().Result)
}

func Test_NewPipelineRun_NotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "notExisting1")
	factory := fake.NewClientFactory()

	// EXERCISE
	examinee, err := NewPipelineRun(run, factory)

	// VERIFY
	assert.NilError(t, err)
	assert.Assert(t, examinee == nil)
}

func Test_NewPipelineRun_Error(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)
	expectedError := fmt.Errorf("expected")
	factory.StewardClientset().PrependReactor("get", "*", fake.NewErrorReactor(expectedError))

	// EXERCISE
	examinee, err := NewPipelineRun(run, factory)

	// VERIFY
	assert.Assert(t, expectedError == err)
	assert.Assert(t, examinee == nil)
}

func Test_pipelineRun_StoreErrorAsMessage(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)
	examinee, err := NewPipelineRun(run, factory)
	examinee.GetStatus().State = api.StateRunning
	assert.NilError(t, err)
	errorToStore := fmt.Errorf("error1")
	message := "message1"

	// EXERCISE
	examinee.StoreErrorAsMessage(errorToStore, message)
	examinee.CommitStatus()

	// VERIFY
	client := factory.StewardV1alpha1().PipelineRuns(ns1)
	run, err = client.Get("foo", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, "ERROR: message1 [PipelineRun{name: foo, namespace: namespace1, state: running}]: error1", run.Status.Message)
}

func Test_pipelineRun_HasDeletionTimestamp_false(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee, err := NewPipelineRun(run, nil)
	assert.NilError(t, err)

	// EXERCISE
	deleted := examinee.HasDeletionTimestamp()

	// VERIFY
	assert.Assert(t, deleted == false)
}

func Test_pipelineRun_HasDeletionTimestamp_true(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	now := metav1.Now()
	run.SetDeletionTimestamp(&now)
	examinee, err := NewPipelineRun(run, nil)
	assert.NilError(t, err)

	// EXERCISE
	deleted := examinee.HasDeletionTimestamp()

	// VERIFY
	assert.Assert(t, deleted == true)
}

func Test_pipelineRun_UpdateMessage_GoodCase(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(run)
	examinee, err := NewPipelineRun(run, factory)
	assert.NilError(t, err)

	// EXERCISE
	examinee.UpdateMessage(message)

	// VERIFY
	assert.Equal(t, message, examinee.GetStatus().Message)
}

func Test_pipelineRun_InitState(t *testing.T) {
	t.Parallel()

	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	creationTimestamp := metav1.Now()
	pipelineRun.ObjectMeta.CreationTimestamp = creationTimestamp
	factory := fake.NewClientFactory(pipelineRun)
	examinee, err := NewPipelineRun(pipelineRun, factory)
	assert.NilError(t, err)

	// EXERCISE
	resultErr := examinee.InitState()

	// VERIFY
	assert.NilError(t, resultErr)

	assert.Equal(t, api.StateNew, examinee.GetStatus().State)
	assert.Equal(t, 0, len(examinee.GetStatus().StateHistory))

	details := examinee.GetStatus().StateDetails
	assert.Equal(t, creationTimestamp, details.StartedAt)
	assert.Equal(t, api.StateNew, details.State)

}

func Test_pipelineRun_InitState_ReturnsErrorIfCalledMultipleTimes(t *testing.T) {
	t.Parallel()

	for _, oldState := range []api.State{
		api.StateNew,
		api.StatePreparing,
		api.StateWaiting,
		api.StateRunning,
		api.StateCleaning,
		api.StateFinished,
	} {

		t.Run(string(oldState), func(t *testing.T) {
			// SETUP
			pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
			creationTimestamp := metav1.Now()
			pipelineRun.ObjectMeta.CreationTimestamp = creationTimestamp
			factory := fake.NewClientFactory(pipelineRun)

			examinee, err := NewPipelineRun(pipelineRun, factory)
			assert.NilError(t, err)
			resultErr := examinee.InitState()
			assert.NilError(t, resultErr)

			resultErr = examinee.UpdateState(oldState, metav1.Now())
			assert.NilError(t, resultErr)

			// EXERCISE
			resultErr = examinee.InitState()

			// VERIFY
			assert.Error(t, resultErr, "Cannot initialize multiple times")
			assert.Equal(t, oldState, examinee.GetStatus().State)
		})
	}
}

func Test_pipelineRun_UpdateState_HasAutomaticInitialization(t *testing.T) {
	t.Parallel()

	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(pipelineRun)
	examinee, err := NewPipelineRun(pipelineRun, factory)
	assert.NilError(t, err)

	// EXERCISE
	resultErr := examinee.UpdateState(api.StatePreparing, metav1.Now())
	assert.NilError(t, resultErr)
	results, retry, resultErr := examinee.CommitStatus()

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Assert(t, retry == false)
	assert.Equal(t, api.StatePreparing, examinee.GetStatus().State)
	assert.Equal(t, 1, len(examinee.GetStatus().StateHistory))
	assert.Equal(t, api.StateNew, examinee.GetStatus().StateHistory[0].State)
	startedAt := examinee.GetStatus().StartedAt
	assert.Assert(t, !startedAt.IsZero())
	assert.Equal(t, *startedAt, examinee.GetStatus().StateHistory[0].FinishedAt)
	assert.Equal(t, api.StateNew, results[0].State)
	assert.Equal(t, *startedAt, results[0].FinishedAt)
}

func Test_pipelineRun_UpdateState_AfterFirstCall(t *testing.T) {
	t.Parallel()

	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	creationTimestamp := metav1.Now()
	pipelineRun.ObjectMeta.CreationTimestamp = creationTimestamp
	factory := fake.NewClientFactory(pipelineRun)
	examinee, err := NewPipelineRun(pipelineRun, factory)
	assert.NilError(t, err)
	err = examinee.InitState()
	assert.NilError(t, err)

	// EXERCISE
	resultErr := examinee.UpdateState(api.StatePreparing, metav1.Now())
	assert.NilError(t, resultErr)
	results, retry, resultErr := examinee.CommitStatus()

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Assert(t, retry == false)
	assert.Equal(t, api.StatePreparing, examinee.GetStatus().State)
	assert.Equal(t, 1, len(examinee.GetStatus().StateHistory))
	assert.Equal(t, api.StateNew, examinee.GetStatus().StateHistory[0].State)
	assert.Equal(t, creationTimestamp, examinee.GetStatus().StateHistory[0].StartedAt)
	startedAt := examinee.GetStatus().StartedAt
	assert.Assert(t, !startedAt.IsZero())
	assert.Equal(t, *startedAt, examinee.GetStatus().StateHistory[0].FinishedAt)

	assert.Equal(t, api.StateNew, results[0].State)
	assert.Equal(t, creationTimestamp, results[0].StartedAt)
	assert.Equal(t, *startedAt, results[0].FinishedAt)
}

func Test_pipelineRun_UpdateState_AfterSecondCall(t *testing.T) {
	t.Parallel()

	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(pipelineRun)
	examinee, err := NewPipelineRun(pipelineRun, factory)
	assert.NilError(t, err)
	err = examinee.InitState()
	assert.NilError(t, err)
	err = examinee.UpdateState(api.StatePreparing, metav1.Now()) // first call
	assert.NilError(t, err)
	factory.Sleep("let time elapse to check timestamps afterwards")

	// EXERCISE
	resultErr := examinee.UpdateState(api.StateRunning, metav1.Now()) // second call
	assert.NilError(t, resultErr)
	results, retry, resultErr := examinee.CommitStatus()

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Assert(t, retry == false)
	status := examinee.GetStatus()
	assert.Equal(t, 2, len(status.StateHistory))
	assert.Equal(t, api.StateNew, examinee.GetStatus().StateHistory[0].State)
	assert.Equal(t, api.StatePreparing, status.StateHistory[1].State)

	start := status.StateHistory[1].StartedAt
	end := status.StateHistory[1].FinishedAt
	assert.Assert(t, !start.IsZero())
	assert.Assert(t, factory.CheckTimeOrder(start, end))

	assert.Equal(t, api.StateNew, results[0].State)
	assert.Equal(t, api.StatePreparing, results[1].State)
	start = results[1].StartedAt
	end = results[1].FinishedAt
	assert.Assert(t, !start.IsZero())
	assert.Assert(t, factory.CheckTimeOrder(start, end))

}

func Test_pipelineRun_UpdateStateToFinished_HistoryIfUpdateStateCalledBefore(t *testing.T) {
	t.Parallel()

	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(pipelineRun)
	examinee, err := NewPipelineRun(pipelineRun, factory)
	assert.NilError(t, err)
	err = examinee.InitState()
	assert.NilError(t, err)
	err = examinee.UpdateState(api.StatePreparing, metav1.Now()) // called before
	assert.NilError(t, err)
	factory.Sleep("let time elapse to check timestamps afterwards")

	// EXERCISE
	examinee.UpdateState(api.StateFinished, metav1.Now())
	_, _, err = examinee.CommitStatus()
	assert.NilError(t, err)

	// VERIFY
	status := examinee.GetStatus()
	assert.Equal(t, 2, len(status.StateHistory))
	assert.Equal(t, api.StatePreparing, status.StateHistory[1].State)

	start := status.StateHistory[1].StartedAt
	end := status.StateHistory[1].FinishedAt
	assert.Assert(t, factory.CheckTimeOrder(start, end))

	assert.Equal(t, api.StateFinished, status.State)
	assert.Equal(t, status.StateDetails.StartedAt, status.StateDetails.FinishedAt)
}

func Test_pipelineRun_UpdateResult(t *testing.T) {
	t.Parallel()

	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(pipelineRun)
	examinee, err := NewPipelineRun(pipelineRun, factory)
	assert.NilError(t, err)
	assert.Assert(t, examinee.GetStatus().FinishedAt.IsZero())

	// EXERCISE
	examinee.UpdateResult(api.ResultSuccess, metav1.Now())

	// VERIFY
	status := examinee.GetStatus()
	assert.Equal(t, api.ResultSuccess, status.Result)
	assert.Assert(t, !examinee.GetStatus().FinishedAt.IsZero())
}
func Test_pipelineRun_GetPipelineRepoServerURL_CorrectURLs(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		url         string
		expectedURL string
	}{
		{url: "http://foo.com/Path", expectedURL: "http://foo.com"},
		{url: "HTTP://foo.com/Path", expectedURL: "http://foo.com"},
		{url: "https://foo.com/Path", expectedURL: "https://foo.com"},
		{url: "HTTPS://foo.com/Path", expectedURL: "https://foo.com"},
		{url: "https://foo.com:1234/Path", expectedURL: "https://foo.com:1234"},
		{url: "http://foo.com:1234/Path", expectedURL: "http://foo.com:1234"},
	} {
		t.Run(test.url, func(t *testing.T) {
			// SETUP
			run := newPipelineRunWithPipelineRepoURL(ns1, run1, test.url)
			r, err := NewPipelineRun(run, nil)
			assert.NilError(t, err)

			// EXERCISE
			url, err := r.GetPipelineRepoServerURL()

			// VERIFY
			assert.NilError(t, err)
			assert.Equal(t, test.expectedURL, url)
		})
	}
}
func Test_pipelineRun_GetPipelineRepoServerURL_WrongURLs(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		url                  string
		expectedErrorPattern string
	}{
		{url: "&:", expectedErrorPattern: `value "&:" of field spec.jenkinsFile.url is invalid \[.*\]: .*`},
		{url: "ftp://foo/bar", expectedErrorPattern: `value "ftp://foo/bar" of field spec.jenkinsFile.url is invalid \[.*\]: scheme not supported: .*`},
	} {
		t.Run(test.url, func(t *testing.T) {
			// SETUP
			run := newPipelineRunWithPipelineRepoURL(ns1, run1, test.url)
			r, err := NewPipelineRun(run, nil)
			assert.NilError(t, err)

			// EXERCISE
			url, err := r.GetPipelineRepoServerURL()

			// VERIFY
			assert.Assert(t, is.Regexp(test.expectedErrorPattern, err.Error()))
			assert.Equal(t, "", url)
		})
	}
}

func Test_pipelineRun_UpdateState_PropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)

	examinee, err := NewPipelineRun(run, factory)
	assert.NilError(t, err)
	err = examinee.InitState()
	assert.NilError(t, err)
	expectedError := fmt.Errorf("expected")
	factory.StewardClientset().PrependReactor("update", "*", fake.NewErrorReactor(expectedError))

	// EXCERCISE
	examinee.UpdateState(api.StateWaiting, metav1.Now())
	_, _, err = examinee.CommitStatus()

	// VERIFY
	assert.Assert(t, err != nil)
}

func Test_pipelineRun_CommitStatus_RetriesOnConflict(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)

	count := 0
	factory.StewardClientset().PrependReactor(
		"update", "pipelineruns",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			if count < 3 {
				count++
				return true, nil, k8serrors.NewConflict(api.Resource("pipelineruns"), "", nil)
			}
			return false, nil, nil
		},
	)

	examinee, err := NewPipelineRun(run, factory)
	assert.NilError(t, err)
	err = examinee.InitState()
	assert.NilError(t, err)
	resultErr := examinee.UpdateState(api.StateWaiting, metav1.Now())
	assert.NilError(t, resultErr)

	// EXCERCISE
	_, _, resultErr = examinee.CommitStatus()

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, examinee.(*pipelineRun).apiObj.Status.State, api.StateWaiting)
	assert.Assert(t, count == 3)
}

func Test_pipelineRun_changeStatusAndUpdateSafely_SetsUpdateResult_IfNoConflict(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)

	updateResultObj := newPipelineRunWithEmptySpec(ns1, "bar")
	factory.StewardClientset().PrependReactor(
		"update", "pipelineruns",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, updateResultObj, nil
		},
	)

	examinee := &pipelineRun{
		apiObj: run,
		copied: false,
		client: factory.StewardV1alpha1().PipelineRuns(ns1),
	}

	changeCallCount := 0
	changeFunc := func(*api.PipelineStatus) (commitRecorderFunc, error) {
		changeCallCount++
		return nil, nil
	}

	// EXCERCISE
	examinee.changeStatusAndStoreForRetry(changeFunc)
	_, _, resultErr := examinee.CommitStatus()

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, examinee.apiObj, updateResultObj)
	assert.Equal(t, examinee.copied, false)
	assert.Equal(t, changeCallCount, 1)
}

func Test_pipelineRun_changeStatusAndUpdateSafely_NoUpdateOnChangeErrorInFirstAttempt(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)

	factory.StewardClientset().PrependReactor(
		"update", "pipelineruns",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			panic("No update expected")
		},
	)

	changeError := fmt.Errorf("ChangeError1")
	changeCallCount := 0
	changeFunc := func(*api.PipelineStatus) (commitRecorderFunc, error) {
		changeCallCount++
		return nil, changeError
	}

	examinee := &pipelineRun{
		apiObj: run,
		copied: false,
		client: factory.StewardV1alpha1().PipelineRuns(ns1),
	}

	// EXCERCISE
	resultErr := examinee.changeStatusAndStoreForRetry(changeFunc)

	// VERIFY
	assert.Error(t, resultErr, changeError.Error())
	assert.Equal(t, changeCallCount, 1)
}

func Test_pipelineRun_changeStatusAndUpdateSafely_SetsUpdateResult_IfConflict(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)

	updateResultObj := newPipelineRunWithEmptySpec(ns1, "bar")
	updateCount := 0
	factory.StewardClientset().PrependReactor(
		"update", "pipelineruns",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			updateCount++
			if updateCount == 1 {
				return true, nil, k8serrors.NewConflict(api.Resource("pipelineruns"), "", nil)
			}
			return true, updateResultObj, nil
		},
	)

	examinee := &pipelineRun{
		apiObj: run,
		copied: false,
		client: factory.StewardV1alpha1().PipelineRuns(ns1),
	}

	changeCallCount := 0
	changeFunc := func(*api.PipelineStatus) (commitRecorderFunc, error) {
		changeCallCount++
		return nil, nil
	}

	// EXCERCISE
	examinee.changeStatusAndStoreForRetry(changeFunc)
	_, _, resultErr := examinee.CommitStatus()

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, examinee.apiObj, updateResultObj)
	assert.Equal(t, examinee.copied, true)
	assert.Equal(t, changeCallCount, 2)
}

func Test_pipelineRun_changeStatusAndUpdateSafely_FailsAfterTooManyConflicts(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)

	expectedRetrySteps := retry.DefaultBackoff.Steps

	updateCount := 0
	errorOnUpdate := k8serrors.NewConflict(api.Resource("pipelineruns"), "", errors.New("error on update"))
	factory.StewardClientset().PrependReactor(
		"update", "pipelineruns",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			if updateCount < expectedRetrySteps*10 {
				updateCount++
				return true, nil, errorOnUpdate
			}
			return false, nil, nil
		},
	)

	examinee := &pipelineRun{
		apiObj: run,
		copied: false,
		client: factory.StewardV1alpha1().PipelineRuns(ns1),
	}

	changeCallCount := 0
	changeFunc := func(*api.PipelineStatus) (commitRecorderFunc, error) {
		changeCallCount++
		return nil, nil
	}

	// EXCERCISE
	examinee.changeStatusAndStoreForRetry(changeFunc)
	_, _, resultErr := examinee.CommitStatus()

	// VERIFY
	assert.Assert(t, errors.Is(resultErr, errorOnUpdate))
	assert.ErrorContains(t, resultErr, "failed to update status")
	assert.ErrorContains(t, resultErr, "error on update")
	assert.Equal(t, updateCount, expectedRetrySteps)
	assert.Equal(t, changeCallCount, expectedRetrySteps)
}

func Test_pipelineRun_changeStatusAndUpdateSafely_ReturnsErrorIfFetchFailed(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)

	factory.StewardClientset().PrependReactor(
		"update", "pipelineruns",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, k8serrors.NewConflict(api.Resource("pipelineruns"), "", errors.New("error on update"))
		},
	)

	errorOnGet := k8serrors.NewForbidden(api.Resource("pipelineruns"), "blah", errors.New("error on get"))
	factory.StewardClientset().PrependReactor(
		"get", "pipelineruns",
		func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errorOnGet
		},
	)

	examinee := &pipelineRun{
		apiObj: run,
		copied: false,
		client: factory.StewardV1alpha1().PipelineRuns(ns1),
	}

	changeCallCount := 0
	changeFunc := func(*api.PipelineStatus) (commitRecorderFunc, error) {
		changeCallCount++
		return nil, nil
	}

	// EXCERCISE
	examinee.changeStatusAndStoreForRetry(changeFunc)
	_, _, resultErr := examinee.CommitStatus()

	// VERIFY
	assert.Assert(t, errors.Is(resultErr, errorOnGet))
	assert.ErrorContains(t, resultErr, "failed to update status")
	assert.ErrorContains(t, resultErr, "failed to fetch pipeline after update conflict")
	assert.ErrorContains(t, resultErr, "error on get")
	assert.Equal(t, changeCallCount, 1)
}

func Test_pipelineRun_CommitStatus_PanicsIfNoClientFactory(t *testing.T) {
	t.Parallel()

	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, run1)
	examinee, err := NewPipelineRun(run, nil /* client factory */)
	assert.NilError(t, err)
	examinee2 := examinee.(*pipelineRun)
	examinee2.changeStatusAndStoreForRetry(func(*api.PipelineStatus) (commitRecorderFunc, error) { /* foo */ return nil, nil })

	// EXERCISE and VERIFY
	assert.Assert(t, cmp.Panics(
		func() {
			examinee2.CommitStatus()
		},
	))
}

func newPipelineRunWithSecret(ns string, name string, secretName string) *api.PipelineRun {
	return fake.PipelineRun(name, ns, api.PipelineSpec{
		Secrets: []string{secretName},
	})
}

func newPipelineRunWithEmptySpec(ns string, name string) *api.PipelineRun {
	return fake.PipelineRun(name, ns, api.PipelineSpec{})
}

func newPipelineRunWithPipelineRepoURL(ns string, name string, url string) *api.PipelineRun {
	return fake.PipelineRun(name, ns, api.PipelineSpec{
		JenkinsFile: api.JenkinsFile{URL: url},
	})
}
