package k8s

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"testing"
)

const message string = "MyMessage"

func Test__pipelineRun_FetchNotExisting__ReturnsNil(t *testing.T) {
	factory := fake.NewClientFactory()
	pipelineRun, err := NewPipelineRunFetcher(factory).ByName(ns1, "NotExisting1")
	assert.Assert(t, pipelineRun == nil)
	assert.NilError(t, err)
}

func Test__pipelineRun_Fetch__ReturnsPipelineRun(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun(ns1, run1))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	assert.Equal(t, run1, r.GetName())
	assert.Equal(t, ns1, r.GetNamespace())
	assert.Equal(t, api.StateUndefined, r.GetStatus().State, "Initial State should be 'StateUndefined'")
	assert.Equal(t, "secret1", r.GetSpec().Secrets[0])
}

func Test__pipelineRun_FetchByKey_ReturnsPipelineRun(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun(ns1, run1))
	key := fake.ObjectKey(run1, ns1)
	r, _ := NewPipelineRunFetcher(factory).ByKey(key)
	assert.Equal(t, run1, r.GetName())
	assert.Equal(t, ns1, r.GetNamespace())
	assert.Equal(t, api.StateUndefined, r.GetStatus().State, "Initial State should be 'StateUndefined'")
	assert.Equal(t, "secret1", r.GetSpec().Secrets[0])
}

func Test__pipelineRun_UpdateMessage__works(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun(ns1, run1))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	r.UpdateState(api.StatePreparing)
	r.UpdateMessage(message)
	assert.Equal(t, message, r.GetStatus().Message)
}

func Test__pipelineRun_calling_UpdateState_Once__yieldsNoHistory(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun(ns1, run1))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	r.UpdateState(api.StatePreparing)
	assert.Equal(t, api.StatePreparing, r.GetStatus().State)
	assert.Equal(t, 0, len(r.GetStatus().StateHistory))
}

func Test__pipelineRun_calling_UpdateState_Twice_yieldsHistoryWithOneEntry(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun(ns1, run1))
	key := fake.ObjectKey(run1, ns1)
	r, _ := NewPipelineRunFetcher(factory).ByKey(key)
	r.UpdateState(api.StatePreparing)
	factory.Sleep("Next State")
	r.UpdateState(api.StateRunning)

	status := r.GetStatus()
	assert.Equal(t, api.StatePreparing, status.StateHistory[0].State)

	start := status.StateHistory[0].StartedAt
	end := status.StateHistory[0].FinishedAt
	assert.Assert(t, factory.CheckTimeOrder(start, end))
	assert.Equal(t, 1, len(status.StateHistory))
}

func Test__pipelineRun_calling_UpdateState_and_FinishState_yieldsHistoryWithOneEntry(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun(ns1, run1))
	key := fake.ObjectKey(run1, ns1)
	r, _ := NewPipelineRunFetcher(factory).ByKey(key)
	r.UpdateState(api.StatePreparing)
	factory.Sleep("Next State")
	r.FinishState()

	status := r.GetStatus()
	assert.Equal(t, api.StatePreparing, status.StateHistory[0].State)

	start := status.StateHistory[0].StartedAt
	end := status.StateHistory[0].FinishedAt
	assert.Assert(t, factory.CheckTimeOrder(start, end))
	assert.Equal(t, 1, len(status.StateHistory))
}

func Test_pipelineRun_GetRepoServerURL_CorrectURLHTTP(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRunWithURL(ns1, run1, "HTTP://foo.com/Path"))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	url, err := r.GetRepoServerURL()
	assert.NilError(t, err)
	assert.Equal(t, "http://foo.com", url)

}

func Test_pipelineRun_GetRepoServerURL_CorrectURL(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRunWithURL(ns1, run1, "https://foo.com/Path"))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	url, err := r.GetRepoServerURL()
	assert.NilError(t, err)
	assert.Equal(t, "https://foo.com", url)

}

func Test_pipelineRun_GetRepoServerURL_CorrectURLWithPort(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRunWithURL(ns1, run1, "https://foo.com:1234/Path"))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	url, err := r.GetRepoServerURL()
	assert.NilError(t, err)
	assert.Equal(t, "https://foo.com:1234", url)

}

func Test_pipelineRun_GetRepoServerURL_WrongUrl(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRunWithURL(ns1, run1, "&:"))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	url, err := r.GetRepoServerURL()
	assert.Assert(t, is.Regexp("failed to parse jenkinsFile.url '&:'.+", err.Error()))
	assert.Equal(t, "", url)
}

func Test_pipelineRun_GetRepoServerURL_WrongScheme(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRunWithURL(ns1, run1, "ftp://foo/bar"))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	url, err := r.GetRepoServerURL()
	assert.Equal(t, "scheme not supported 'ftp'", err.Error())
	assert.Equal(t, "", url)

}

func newPipelineRun(ns string, name string) *api.PipelineRun {
	return fake.PipelineRun(name, ns, api.PipelineSpec{
		Secrets: []string{"secret1"},
	})
}

func newPipelineRunWithURL(ns string, name string, url string) *api.PipelineRun {
	return fake.PipelineRun(name, ns, api.PipelineSpec{
		Secrets:     []string{"secret1"},
		JenkinsFile: api.JenkinsFile{URL: url},
	})
}
