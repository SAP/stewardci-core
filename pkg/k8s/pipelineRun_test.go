package k8s

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
)

const message string = "MyMessage"

func Test__FetchNotExisting__ReturnsNil(t *testing.T) {
	factory := fake.NewClientFactory()
	pipelineRun, err := NewPipelineRunFetcher(factory).ByName(ns1, "NotExisting1")
	assert.Assert(t, pipelineRun == nil)
	assert.NilError(t, err)
}

func Test__Fetch__ReturnsPipelineRun(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun())
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	assert.Equal(t, run1, r.GetName())
	assert.Equal(t, ns1, r.GetNamespace())
	assert.Equal(t, api.StateUndefined, r.GetStatus().State, "Initial State should be 'StateUndefined'")
	assert.Equal(t, "secret1", r.GetSpec().Secrets[0])
}

func Test__FetchByKey_ReturnsPipelineRun(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun())
	key := fake.ObjectKey(run1, ns1)
	r, _ := NewPipelineRunFetcher(factory).ByKey(key)
	assert.Equal(t, run1, r.GetName())
	assert.Equal(t, ns1, r.GetNamespace())
	assert.Equal(t, api.StateUndefined, r.GetStatus().State, "Initial State should be 'StateUndefined'")
	assert.Equal(t, "secret1", r.GetSpec().Secrets[0])
}

func Test__UpdateMessage__works(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun())
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	r.UpdateState(api.StatePreparing)
	r.UpdateMessage(message)
	assert.Equal(t, message, r.GetStatus().Message)
}

func Test__calling_UpdateState_Once__yieldsNoHistory(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun())
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	r.UpdateState(api.StatePreparing)
	assert.Equal(t, api.StatePreparing, r.GetStatus().State)
	assert.Equal(t, 0, len(r.GetStatus().StateHistory))
}

func Test__calling_UpdateState_Twice_yieldsHistoryWithOneEntry(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun())
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

func Test__calling_UpdateState_and_FinishState_yieldsHistoryWithOneEntry(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRun())
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

func Test_GetBaseRepo_CorrectURL(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRunWithURL("https://github.com/SAP"))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	url, err := r.GetRepoBaseURL()
	assert.Equal(t, "https://github.com", url)
	assert.NilError(t, err)
}

func Test_GetBaseRepo_WrongUrl(t *testing.T) {
	factory := fake.NewClientFactory(newPipelineRunWithURL("&:"))
	r, _ := NewPipelineRunFetcher(factory).ByName(ns1, run1)
	url, err := r.GetRepoBaseURL()
	assert.Equal(t, "", url)
	assert.Equal(t, "Failed to parse JenkinsFile.URL '&:': parse &:: first path segment in URL cannot contain colon", err.Error())
}

func newPipelineRun() *api.PipelineRun {
	return fake.PipelineRun(run1, ns1, api.PipelineSpec{
		Secrets: []string{"secret1"},
	})
}

func newPipelineRunWithURL(url string) *api.PipelineRun {
	return fake.PipelineRun(run1, ns1, api.PipelineSpec{
		Secrets:     []string{"secret1"},
		JenkinsFile: api.JenkinsFile{URL: url},
	})
}
