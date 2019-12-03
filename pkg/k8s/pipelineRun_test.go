package k8s

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const message string = "MyMessage"

func Test_pipelineRunFetcher_ByName_NotExisting(t *testing.T) {
	// SETUP
	factory := fake.NewClientFactory()
	examinee := NewPipelineRunFetcher(factory)

	// EXERCISE
	pipelineRun, resultErr := examinee.ByName(ns1, "NotExisting1")

	// VERIFY
	assert.Assert(t, pipelineRun == nil)
	assert.NilError(t, resultErr)
}

func Test_pipelineRunFetcher_ByName_GoodCase(t *testing.T) {
	// SETUP
	const (
		secretName = "secret1"
	)

	factory := fake.NewClientFactory(
		newPipelineRunWithSecret(ns1, run1, secretName),
	)
	examinee := NewPipelineRunFetcher(factory)

	// EXERCISE
	resultObj, resultErr := examinee.ByName(ns1, run1)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, run1, resultObj.GetName())
	assert.Equal(t, ns1, resultObj.GetNamespace())
	assert.Equal(t, api.StateUndefined, resultObj.GetStatus().State, "Initial State should be 'StateUndefined'")
	assert.Equal(t, secretName, resultObj.GetSpec().Secrets[0])
}

func Test_pipelineRunFetcher_ByKey_GoodCase(t *testing.T) {
	// SETUP
	const (
		secretName = "secret1"
	)

	factory := fake.NewClientFactory(
		newPipelineRunWithSecret(ns1, run1, secretName),
	)
	key := fake.ObjectKey(run1, ns1)
	examinee := NewPipelineRunFetcher(factory)

	// EXERCISE
	resultObj, resultErr := examinee.ByKey(key)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, run1, resultObj.GetName())
	assert.Equal(t, ns1, resultObj.GetNamespace())
	assert.Equal(t, api.StateUndefined, resultObj.GetStatus().State, "Initial State should be 'StateUndefined'")
	assert.Equal(t, secretName, resultObj.GetSpec().Secrets[0])
}

func Test_pipelineRun_UpdateMessage_GoodCase(t *testing.T) {
	// SETUP
	factory := fake.NewClientFactory(
		newPipelineRunWithEmptySpec(ns1, run1),
	)
	examinee, err := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	assert.NilError(t, err)
	examinee.UpdateState(api.StatePreparing)

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
	examinee, err := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	assert.NilError(t, err)

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
	factory := fake.NewClientFactory(
		newPipelineRunWithEmptySpec(ns1, run1),
	)
	key := fake.ObjectKey(run1, ns1)
	examinee, err := NewPipelineRunFetcher(factory).ByKey(key)
	assert.NilError(t, err)

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
	factory := fake.NewClientFactory(
		newPipelineRunWithEmptySpec(ns1, run1),
	)
	key := fake.ObjectKey(run1, ns1)
	examinee, err := NewPipelineRunFetcher(factory).ByKey(key)
	assert.NilError(t, err)

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
	factory := fake.NewClientFactory(
		newPipelineRunWithEmptySpec(ns1, run1),
	)
	key := fake.ObjectKey(run1, ns1)
	examinee, err := NewPipelineRunFetcher(factory).ByKey(key)
	assert.NilError(t, err)
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
			factory := fake.NewClientFactory(newPipelineRunWithPipelineRepoURL(ns1, run1, test.url))
			r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
			url, err := r.GetPipelineRepoServerURL()
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
			factory := fake.NewClientFactory(newPipelineRunWithPipelineRepoURL(ns1, run1, test.url))
			r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
			url, err := r.GetPipelineRepoServerURL()
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
