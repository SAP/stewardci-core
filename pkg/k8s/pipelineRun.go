package k8s

import (
	"fmt"
	"log"
	"net/url"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	utils "github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// PipelineRun is a wrapper for the K8s PipelineRun resource
type PipelineRun interface {
	GetStatus() *api.PipelineStatus
	GetSpec() *api.PipelineSpec
	GetName() string
	GetKey() string
	GetRunNamespace() string
	GetNamespace() string
	GetPipelineRepoServerURL() (string, error)
	HasDeletionTimestamp() bool
	AddFinalizer() error
	DeleteFinalizerIfExists() error
	UpdateState(api.State) (*api.StateItem, error)
	FinishState() (*api.StateItem, error)
	UpdateResult(api.Result) error
	UpdateContainer(*corev1.ContainerState) error
	StoreErrorAsMessage(error, string) error
	UpdateRunNamespace(string) error
	UpdateMessage(string) error
	UpdateLog()
}

type pipelineRun struct {
	client  stewardv1alpha1.PipelineRunInterface
	fetcher PipelineRunFetcher
	apiObj  *api.PipelineRun
}

// NewPipelineRun creates a managed pipeline run object
func NewPipelineRun(apiObj *api.PipelineRun, fetcher PipelineRunFetcher, factory ClientFactory) PipelineRun {
	result := &pipelineRun{
		fetcher: fetcher,
		apiObj:  apiObj,
	}
	if factory != nil {
		result.client = factory.StewardV1alpha1().PipelineRuns(apiObj.GetNamespace())
	}
	return result
}

func (r *pipelineRun) update() error {
	result, err := r.client.Update(r.apiObj)
	if err == nil {
		r.apiObj = result
	}
	return err
}

// GetRunNamespace returns the namespace in which the build takes place
func (r *pipelineRun) GetRunNamespace() string {
	return r.apiObj.Status.Namespace
}

// GetKey returns the key of the pipelineRun
func (r *pipelineRun) GetKey() string {
	key, _ := cache.MetaNamespaceKeyFunc(r.apiObj)
	return key
}

// GetNamespace returns the namespace of the underlying pipelineRun object
func (r *pipelineRun) GetNamespace() string {
	return r.apiObj.GetNamespace()
}

// GetPipelineRepoServerURL returns the server hosting the Jenkinsfile repository
func (r *pipelineRun) GetPipelineRepoServerURL() (string, error) {
	urlString := r.GetSpec().JenkinsFile.URL
	repoURL, err := url.Parse(urlString)
	if err != nil {
		return "", errors.Wrapf(err, "value %q of field spec.jenkinsFile.url is invalid", urlString)
	}
	if !(repoURL.Scheme == "http") && !(repoURL.Scheme == "https") {
		return "", fmt.Errorf("value %q of field spec.jenkinsFile.url is invalid: scheme not supported: %q", urlString, repoURL.Scheme)
	}
	return fmt.Sprintf("%s://%s", repoURL.Scheme, repoURL.Host), nil
}

func (r *pipelineRun) GetName() string {
	return r.apiObj.GetName()
}

// GetStatus return the Status
func (r *pipelineRun) GetStatus() *api.PipelineStatus {
	return &r.apiObj.Status
}

// GetSpec return the spec part of the PipelineRun resource
func (r *pipelineRun) GetSpec() *api.PipelineSpec {
	return &r.apiObj.Spec
}

// UpdateState set end time of current (defined) state (A) and store it to the history.
// if no current state is defined a new state (A) with cretiontime of the pipelinerun as start time is created.
// It also creates a new current state (B) with start time.
// Returns the state details of state A
func (r *pipelineRun) UpdateState(state api.State) (*api.StateItem, error) {
	log.Printf("New State: %s", state)
	now := metav1.Now()
	oldstate, err := r.FinishState()
	if err != nil {
		return nil, err
	}
	newState := api.StateItem{State: state, StartedAt: now}
	r.apiObj.Status.StateDetails = newState
	r.apiObj.Status.State = state
	return oldstate, r.updateStatus()
}

// FinishState set end time stamp of the current (defined) state and add it to the history
// If no current state is defined a new state (A) with creation time of the PipelineRun as start time is created.
// Returns the state details
func (r *pipelineRun) FinishState() (*api.StateItem, error) {
	state := r.apiObj.Status.StateDetails
	now := metav1.Now()
	if state.State == api.StateUndefined {
		state.State = api.StateNew
		state.StartedAt = r.apiObj.ObjectMeta.CreationTimestamp
		r.apiObj.Status.StartedAt = &now
	}
	state.FinishedAt = now
	his := r.apiObj.Status.StateHistory
	his = append(his, state)
	r.apiObj.Status.StateHistory = his
	return &state, r.updateStatus()
}

// UpdateResult of the pipeline run
func (r *pipelineRun) UpdateResult(result api.Result) error {
	r.apiObj.Status.Result = result
	now := metav1.Now()
	r.apiObj.Status.FinishedAt = &now
	return r.updateStatus()
}

// UpdateContainer ...
func (r *pipelineRun) UpdateContainer(c *corev1.ContainerState) error {
	if c == nil {
		return nil
	}
	r.apiObj.Status.Container = *c
	return r.updateStatus()
}

// StoreErrorAsMessage stores the error as message in the status
func (r *pipelineRun) StoreErrorAsMessage(err error, message string) error {
	if err != nil {
		text := fmt.Sprintf("ERROR: %s (%s - status:%s): %s", utils.Trim(message), r.GetName(), string(r.GetStatus().State), err.Error())
		log.Printf(text)
		return r.UpdateMessage(text)
	}
	return nil
}

// UpdateMessage stores string as message in the status
func (r *pipelineRun) UpdateMessage(message string) error {
	old := r.apiObj.Status.Message
	if old != "" {
		his := r.apiObj.Status.History
		his = append(his, old)
		r.apiObj.Status.History = his
	}
	r.apiObj.Status.Message = utils.Trim(message)
	r.apiObj.Status.MessageShort = utils.ShortenMessage(message, 100)
	return r.updateStatus()
}

// UpdateRunNamespace overrides the namespace in which the builds happens
func (r *pipelineRun) UpdateRunNamespace(ns string) error {
	r.apiObj.Status.Namespace = ns
	return r.updateStatus()
}

// UpdateLog ...
func (r *pipelineRun) UpdateLog() {
	if r.apiObj.Status.Namespace != "" {
		r.apiObj.Status.LogURL = "dummy://foo"
		r.updateStatus()
	}
}

//HasDeletionTimestamp returns true if deletion timestamp is set
func (r *pipelineRun) HasDeletionTimestamp() bool {
	return !r.apiObj.ObjectMeta.DeletionTimestamp.IsZero()
}

// AddFinalizer adds a finalizer to pipeline run
func (r *pipelineRun) AddFinalizer() error {
	changed, finalizerList := utils.AddStringIfMissing(r.apiObj.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		r.apiObj.ObjectMeta.Finalizers = finalizerList
		return r.update()
	}
	return nil
}

// DeleteFinalizerIfExists deletes a finalizer from pipeline run
func (r *pipelineRun) DeleteFinalizerIfExists() error {
	changed, finalizerList := utils.RemoveString(r.apiObj.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		r.apiObj.ObjectMeta.Finalizers = finalizerList
		return r.update()
	}
	return nil
}

func (r *pipelineRun) updateStatus() error {
	pipelineRun, err := r.fetcher.ByName(r.apiObj.GetNamespace(), r.apiObj.GetName())
	if err != nil {
		return err
	}
	pipelineRun.Status = r.apiObj.Status
	result, err := r.client.UpdateStatus(pipelineRun)
	if err != nil {
		return errors.Wrap(err,
			fmt.Sprintf("Failed to update status of PipelineRun '%s' in namespace '%s'", r.apiObj.GetName(), r.apiObj.GetNamespace()))
	}
	r.apiObj = result
	return nil
}
