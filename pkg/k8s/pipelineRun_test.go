package k8s

import (
	"fmt"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const message string = "MyMessage"

func Test_GetRunNamespace(t *testing.T) {
	//SETUP
	run := &api.PipelineRun{
		Status: api.PipelineStatus{
			Namespace: "foo",
		},
	}
	examinee := NewPipelineRun(run, nil)
	// EXERCISE
	ns := examinee.GetRunNamespace()
	// VERIFY
	assert.Equal(t, "foo", ns)
}

func Test_GetKey(t *testing.T) {
	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee := NewPipelineRun(run, nil)
	// EXERCISE
	key := examinee.GetKey()
	// VERIFY
	assert.Equal(t, "ns1/foo", key)
}

func Test_GetNamespace(t *testing.T) {
	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee := NewPipelineRun(run, nil)
	// EXERCISE
	key := examinee.GetNamespace()
	// VERIFY
	assert.Equal(t, "ns1", key)
}

func Test_GetName(t *testing.T) {
	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee := NewPipelineRun(run, nil)
	// EXERCISE
	key := examinee.GetName()
	// VERIFY
	assert.Equal(t, "foo", key)
}

func Test_StoreErrorAsMessage(t *testing.T) {
	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, "foo")
	factory := fake.NewClientFactory(run)
	examinee := NewPipelineRun(run, factory)
	errorToStore := fmt.Errorf("error1")
	message := "message1"
	// EXERCISE
	examinee.StoreErrorAsMessage(errorToStore, message)
	// VERIFY
	client := factory.StewardV1alpha1().PipelineRuns(ns1)
	run, err := client.Get("foo", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, "ERROR: message1 (foo - status:): error1", run.Status.Message)
}

func Test_HasDeletionTimestamp_false(t *testing.T) {
	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	examinee := NewPipelineRun(run, nil)
	// EXERCISE
	deleted := examinee.HasDeletionTimestamp()
	// VERIFY
	assert.Assert(t, deleted == false)
}

func Test_HasDeletionTimestamp_true(t *testing.T) {
	// SETUP
	run := newPipelineRunWithEmptySpec("ns1", "foo")
	now := metav1.Now()
	run.SetDeletionTimestamp(&now)
	examinee := NewPipelineRun(run, nil)
	// EXERCISE
	deleted := examinee.HasDeletionTimestamp()
	// VERIFY
	assert.Assert(t, deleted == true)
}

func Test_pipelineRun_UpdateMessage_GoodCase(t *testing.T) {
	// SETUP
	run := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(run)
	examinee := NewPipelineRun(run, factory)

	// EXERCISE
	examinee.UpdateMessage(message)

	// VERIFY
	assert.Equal(t, message, examinee.GetStatus().Message)
}

func Test_pipelineRun_UpdateState_AfterFirstCall(t *testing.T) {
	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	creationTimestamp := metav1.Now()
	pipelineRun.ObjectMeta.CreationTimestamp = creationTimestamp
	factory := fake.NewClientFactory(pipelineRun)
	examinee := NewPipelineRun(pipelineRun, factory)

	// EXERCISE
	examinee.UpdateState(api.StatePreparing)

	// VERIFY
	assert.Equal(t, api.StatePreparing, examinee.GetStatus().State)
	assert.Equal(t, 1, len(examinee.GetStatus().StateHistory))
	assert.Equal(t, api.StateNew, examinee.GetStatus().StateHistory[0].State)
	assert.Equal(t, creationTimestamp, examinee.GetStatus().StateHistory[0].StartedAt)
	startedAt := examinee.GetStatus().StartedAt
	assert.Assert(t, !startedAt.IsZero())
	assert.Equal(t, *startedAt, examinee.GetStatus().StateHistory[0].FinishedAt)
}

func Test_pipelineRun_UpdateState_AfterSecondCall(t *testing.T) {
	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(pipelineRun)
	examinee := NewPipelineRun(pipelineRun, factory)

	examinee.UpdateState(api.StatePreparing) // first call
	factory.Sleep("let time elapse to check timestamps afterwards")

	// EXERCISE
	examinee.UpdateState(api.StateRunning) // second call

	// VERIFY
	status := examinee.GetStatus()
	assert.Equal(t, 2, len(status.StateHistory))
	assert.Equal(t, api.StateNew, examinee.GetStatus().StateHistory[0].State)
	assert.Equal(t, api.StatePreparing, status.StateHistory[1].State)

	start := status.StateHistory[1].StartedAt
	end := status.StateHistory[1].FinishedAt
	assert.Assert(t, factory.CheckTimeOrder(start, end))
}

func Test_pipelineRun_FinishState_HistoryIfUpdateStateCalledBefore(t *testing.T) {
	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(pipelineRun)
	examinee := NewPipelineRun(pipelineRun, factory)

	examinee.UpdateState(api.StatePreparing) // called before
	factory.Sleep("let time elapse to check timestamps afterwards")

	// EXERCISE
	examinee.FinishState()

	// VERIFY
	status := examinee.GetStatus()
	assert.Equal(t, 2, len(status.StateHistory))
	assert.Equal(t, api.StatePreparing, status.StateHistory[1].State)

	start := status.StateHistory[1].StartedAt
	end := status.StateHistory[1].FinishedAt
	assert.Assert(t, factory.CheckTimeOrder(start, end))
}

func Test_pipelineRun_UpdateResult(t *testing.T) {
	// SETUP
	pipelineRun := newPipelineRunWithEmptySpec(ns1, run1)
	factory := fake.NewClientFactory(pipelineRun)
	examinee := NewPipelineRun(pipelineRun, factory)

	assert.Assert(t, examinee.GetStatus().FinishedAt.IsZero())
	// EXERCISE
	examinee.UpdateResult(api.ResultSuccess)
	// VERIFY
	status := examinee.GetStatus()
	assert.Equal(t, api.ResultSuccess, status.Result)
	assert.Assert(t, !examinee.GetStatus().FinishedAt.IsZero())

}

func Test_pipelineRun_GetPipelineRepoServerURL_CorrectURLs(t *testing.T) {
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
			r := NewPipelineRun(run, nil)
			// EXERCISE
			url, err := r.GetPipelineRepoServerURL()
			// VERIFY
			assert.NilError(t, err)
			assert.Equal(t, test.expectedURL, url)
		})
	}
}
func Test_pipelineRun_GetPipelineRepoServerURL_WrongURLs(t *testing.T) {
	for _, test := range []struct {
		url                  string
		expectedErrorPattern string
	}{
		{url: "&:", expectedErrorPattern: `value "&:" of field spec.jenkinsFile.url is invalid: .*`},
		{url: "ftp://foo/bar", expectedErrorPattern: `value "ftp://foo/bar" of field spec.jenkinsFile.url is invalid: scheme not supported: .*`},
	} {
		t.Run(test.url, func(t *testing.T) {
			// SETUP
			run := newPipelineRunWithPipelineRepoURL(ns1, run1, test.url)
			r := NewPipelineRun(run, nil)
			// EXERCISE
			url, err := r.GetPipelineRepoServerURL()
			// VERIFY
			assert.Assert(t, is.Regexp(test.expectedErrorPattern, err.Error()))
			assert.Equal(t, "", url)
		})
	}
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
