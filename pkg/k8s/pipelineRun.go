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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	namespace string
	client    stewardv1alpha1.PipelineRunInterface
	name      string
	cached    *api.PipelineRun
}

// PipelineRunFetcher has methods to fetch PipelineRun objects from Kubernetes
type PipelineRunFetcher interface {
	ByName(namespace string, name string) (PipelineRun, error)
	ByKey(key string) (PipelineRun, error)
}

type pipelineRunFetcher struct {
	factory ClientFactory
}

// NewPipelineRunFetcher returns an operative implementation of PipelineRunFetcher
func NewPipelineRunFetcher(factory ClientFactory) PipelineRunFetcher {
	return &pipelineRunFetcher{factory: factory}
}

// ByName fetches PipelineRun resource from Kubernetes by name and namespace
// Return nil,nil if specified pipeline does not exist
func (rf *pipelineRunFetcher) ByName(namespace string, name string) (PipelineRun, error) {
	client := rf.factory.StewardV1alpha1().PipelineRuns(namespace)
	result := &pipelineRun{client: client, name: name, namespace: namespace}
	var err error
	result.cached, err = result.fetch()
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err,
			fmt.Sprintf("Failed to fetch PipelineRun '%s' in namespace '%s'", name, namespace))
	}
	return result, nil
}

// ByKey fetches PipelineRun resource from Kubernetes
// Return nil,nil if pipeline with key does not exist
func (rf *pipelineRunFetcher) ByKey(key string) (PipelineRun, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return &pipelineRun{}, err
	}
	return rf.ByName(namespace, name)
}

func (r *pipelineRun) fetch() (*api.PipelineRun, error) {
	return r.client.Get(r.name, metav1.GetOptions{})
}

func (r *pipelineRun) update() error {
	result, err := r.client.Update(r.cached)
	if err == nil {
		r.cached = result
	}
	return err
}

// GetRunNamespace returns the namespace in which the build takes place
func (r *pipelineRun) GetRunNamespace() string {
	return r.cached.Status.Namespace
}

// GetKey returns the key of the pipelineRun
func (r *pipelineRun) GetKey() string {
	key, _ := cache.MetaNamespaceKeyFunc(r.cached)
	return key
}

// GetNamespace returns the namespace of the underlying pipelineRun object
func (r *pipelineRun) GetNamespace() string {
	return r.cached.GetNamespace()
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
	return r.name
}

// GetStatus return the Status
func (r *pipelineRun) GetStatus() *api.PipelineStatus {
	return &r.cached.Status
}

// GetSpec return the spec part of the PipelineRun resource
func (r *pipelineRun) GetSpec() *api.PipelineSpec {
	return &r.cached.Spec
}

func (r *pipelineRun) UpdateState(state api.State) (*api.StateItem, error) {
	// UpdateState set end time of current (defined) state (A) and store it to the history.
	// if no current state is defined a new pickup state (A) with cretiontime of the pipelinerun as start time is created.
	// It also creates a new current state (B) with start time.
	// Returns the state details of state A
	log.Printf("New State: %s", state)
	now := metav1.Now()
	oldstate, err := r.FinishState()
	if err != nil {
		return nil, err
	}
	newState := api.StateItem{State: state, StartedAt: now}
	r.cached.Status.StateDetails = newState
	r.cached.Status.State = state
	return oldstate, r.updateStatus()
}

// FinishState set end time stamp of the current (defined) state and add it to the history
// if no current state is defined a new pickup state (A) with cretiontime of the pipelinerun as start time is created.
// Returns the state details
func (r *pipelineRun) FinishState() (*api.StateItem, error) {
	state := r.cached.Status.StateDetails
	now := metav1.Now()
	if state.State == api.StateUndefined {
		state.State = api.StatePickup
		state.StartedAt = r.cached.ObjectMeta.CreationTimestamp
		r.cached.Status.StartedAt = now
	}
	state.FinishedAt = now
	his := r.cached.Status.StateHistory
	his = append(his, state)
	r.cached.Status.StateHistory = his
	return &state, r.updateStatus()
}

// UpdateResult of the pipeline run
func (r *pipelineRun) UpdateResult(result api.Result) error {
	r.cached.Status.Result = result
	r.cached.Status.FinishedAt = metav1.Now()
	return r.updateStatus()
}

// UpdateContainer ...
func (r *pipelineRun) UpdateContainer(c *corev1.ContainerState) error {
	if c == nil {
		return nil
	}
	r.cached.Status.Container = *c
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
	old := r.cached.Status.Message
	if old != "" {
		his := r.cached.Status.History
		his = append(his, old)
		r.cached.Status.History = his
	}
	r.cached.Status.Message = utils.Trim(message)
	r.cached.Status.MessageShort = utils.ShortenMessage(message, 100)
	return r.updateStatus()
}

// UpdateRunNamespace overrides the namespace in which the builds happens
func (r *pipelineRun) UpdateRunNamespace(ns string) error {
	r.cached.Status.Namespace = ns
	return r.updateStatus()
}

// UpdateLog ...
func (r *pipelineRun) UpdateLog() {
	if r.cached.Status.Namespace != "" {
		r.cached.Status.LogURL = "dummy://foo"
		r.updateStatus()
	}
}

//HasDeletionTimestamp returns true if deletion timestamp is set
func (r *pipelineRun) HasDeletionTimestamp() bool {
	return !r.cached.ObjectMeta.DeletionTimestamp.IsZero()
}

// AddFinalizer adds a finalizer to pipeline run
func (r *pipelineRun) AddFinalizer() error {
	changed, finalizerList := utils.AddStringIfMissing(r.cached.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		r.cached.ObjectMeta.Finalizers = finalizerList
		return r.update()
	}
	return nil
}

// DeleteFinalizerIfExists deletes a finalizer from pipeline run
func (r *pipelineRun) DeleteFinalizerIfExists() error {
	changed, finalizerList := utils.RemoveString(r.cached.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		r.cached.ObjectMeta.Finalizers = finalizerList
		return r.update()
	}
	return nil
}

func (r *pipelineRun) updateStatus() error {
	pipelineRun, err := r.fetch()
	if err != nil {
		return err
	}
	pipelineRun.Status = r.cached.Status
	result, err := r.client.UpdateStatus(pipelineRun)
	if err != nil {
		return errors.Wrap(err,
			fmt.Sprintf("Failed to update status of PipelineRun '%s' in namespace '%s'", r.name, r.namespace))
	}
	r.cached = result
	return nil
}
