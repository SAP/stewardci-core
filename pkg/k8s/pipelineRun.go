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
	fmt.Stringer
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
	UpdateResult(api.Result) error
	UpdateContainer(*corev1.ContainerState) error
	StoreErrorAsMessage(error, string) error
	UpdateRunNamespace(string) error
	UpdateMessage(string) error
}

type pipelineRun struct {
	client stewardv1alpha1.PipelineRunInterface
	apiObj *api.PipelineRun
	copied bool
}

// NewPipelineRun creates a managed pipeline run object.
// If a factory is provided a new version of the pipelinerun is fetched.
// All changes are done on the fetched object.
// If no pipeline run can be found matching the apiObj, nil,nil is returned.
// An error is only returned if a Get for the pipelinerun returns an error other than a NotFound error.
// If you call with factory nil you can only use the Get* functions
// If you use functions changing the pipeline run without factroy set you will get an error.
// The provided PipelineRun object is never modified and copied as late as possible.
func NewPipelineRun(apiObj *api.PipelineRun, factory ClientFactory) (PipelineRun, error) {
	if factory == nil {
		return &pipelineRun{
			apiObj: apiObj,
			copied: false,
		}, nil
	}
	client := factory.StewardV1alpha1().PipelineRuns(apiObj.GetNamespace())
	obj, err := client.Get(apiObj.GetName(), metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &pipelineRun{
		apiObj: obj,
		copied: true,
		client: client,
	}, nil
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
		return "", errors.Wrapf(err, "value %q of field spec.jenkinsFile.url is invalid [%s]", urlString, r.String())
	}
	if !(repoURL.Scheme == "http") && !(repoURL.Scheme == "https") {
		return "", fmt.Errorf("value %q of field spec.jenkinsFile.url is invalid [%s]: scheme not supported: %q", urlString, r.String(), repoURL.Scheme)
	}
	return fmt.Sprintf("%s://%s", repoURL.Scheme, repoURL.Host), nil
}

func (r *pipelineRun) GetName() string {
	return r.apiObj.GetName()
}

// GetStatus return the Status
// the returned PipelineStatus MUST NOT be modified
// use the prodided Update* functions instead
func (r *pipelineRun) GetStatus() *api.PipelineStatus {
	return &r.apiObj.Status
}

// GetSpec return the spec part of the PipelineRun resource
// the returned PipelineSpec MUST NOT be modified
func (r *pipelineRun) GetSpec() *api.PipelineSpec {
	return &r.apiObj.Spec
}

// UpdateState set end time of current (defined) state (A) and store it to the history.
// if no current state is defined a new state (A) with cretiontime of the pipelinerun as start time is created.
// It also creates a new current state (B) with start time.
// Returns the state details of state A
func (r *pipelineRun) UpdateState(state api.State) (*api.StateItem, error) {
	r.ensureCopy()
	log.Printf("Update State to %s [%s]", state, r.String())
	now := metav1.Now()
	oldstate := r.finishCurrentState()
	newState := api.StateItem{State: state, StartedAt: now}
	if state == api.StateFinished {
		newState.FinishedAt = now
	}
	return oldstate, r.changeStatusAndUpdateSafely(func() {
		r.apiObj.Status.StateDetails = newState
		r.apiObj.Status.State = state
	})
}

// String returns the full qualified name of the pipeline run
func (r *pipelineRun) String() string {
	return fmt.Sprintf("PipelineRun{name: %s, namespace: %s, state: %s}", r.GetName(), r.GetNamespace(), string(r.GetStatus().State))
}

func (r *pipelineRun) finishCurrentState() *api.StateItem {
	r.ensureCopy()
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
	r.apiObj.Status.StateDetails = state
	return &state
}

// UpdateResult of the pipeline run
func (r *pipelineRun) UpdateResult(result api.Result) error {
	r.ensureCopy()
	return r.changeStatusAndUpdateSafely(func() {
		r.apiObj.Status.Result = result
		now := metav1.Now()
		r.apiObj.Status.FinishedAt = &now
	})
}

// UpdateContainer ...
func (r *pipelineRun) UpdateContainer(c *corev1.ContainerState) error {
	if c == nil {
		return nil
	}
	r.ensureCopy()
	return r.changeStatusAndUpdateSafely(func() {
		r.apiObj.Status.Container = *c
	})
}

// StoreErrorAsMessage stores the error as message in the status
func (r *pipelineRun) StoreErrorAsMessage(err error, message string) error {
	if err != nil {
		text := fmt.Sprintf("ERROR: %s [%s]: %s", utils.Trim(message), r.String(), err.Error())
		log.Printf(text)
		return r.UpdateMessage(text)
	}
	return nil
}

// UpdateMessage stores string as message in the status
func (r *pipelineRun) UpdateMessage(message string) error {
	r.ensureCopy()

	return r.changeStatusAndUpdateSafely(func() {
		old := r.apiObj.Status.Message
		if old != "" {
			his := r.apiObj.Status.History
			his = append(his, old)
			r.apiObj.Status.History = his
		}
		r.apiObj.Status.Message = utils.Trim(message)
		r.apiObj.Status.MessageShort = utils.ShortenMessage(message, 100)
	})
}

// UpdateRunNamespace overrides the namespace in which the builds happens
func (r *pipelineRun) UpdateRunNamespace(ns string) error {
	r.ensureCopy()
	return r.changeStatusAndUpdateSafely(func() {
		r.apiObj.Status.Namespace = ns
	})
}

//HasDeletionTimestamp returns true if deletion timestamp is set
func (r *pipelineRun) HasDeletionTimestamp() bool {
	return !r.apiObj.ObjectMeta.DeletionTimestamp.IsZero()
}

// AddFinalizer adds a finalizer to pipeline run
func (r *pipelineRun) AddFinalizer() error {
	changed, finalizerList := utils.AddStringIfMissing(r.apiObj.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		r.updateFinalizers(finalizerList)
	}
	return nil
}

// DeleteFinalizerIfExists deletes a finalizer from pipeline run
func (r *pipelineRun) DeleteFinalizerIfExists() error {
	changed, finalizerList := utils.RemoveString(r.apiObj.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		return r.updateFinalizers(finalizerList)
	}
	return nil
}

func (r *pipelineRun) updateFinalizers(finalizerList []string) error {
	if r.client == nil {
		return fmt.Errorf("No factory provided to store updates [%s]", r.String())
	}
	r.ensureCopy()
	r.apiObj.ObjectMeta.Finalizers = finalizerList
	result, err := r.client.Update(r.apiObj)
	if err != nil {
		return errors.Wrap(err,
			fmt.Sprintf("Failed to update finalizers [%s]", r.String()))
	}
	r.apiObj = result
	return nil
}

// changeStatusAndUpdateSafely executes the change encapsuled in the function parameter "change"
// and updates the Kubernetes object afterwards.
// If the updated fails with "conflict", the object is fechted again, the change is redone and another update try is made.
func (r *pipelineRun) changeStatusAndUpdateSafely(change func()) error {
	if r.client == nil {
		return fmt.Errorf("No factory provided to store updates [%s]", r.String())
	}
	var result *api.PipelineRun
	for { // retry loop
		var err error

		change()
		result, err = r.client.UpdateStatus(r.apiObj)
		if err != nil {
			break // success
		} else {
			if k8serrors.IsConflict(err) {
				log.Printf(
					"retrying update of pipeline run %q in namespace %q"+
						" after resource version conflict",
					r.apiObj.Name, r.apiObj.Namespace,
				)

				new, err := r.client.Get(r.apiObj.GetName(), metav1.GetOptions{})
				if err != nil {
					return errors.Wrap(err,
						"failed to refetch pipeline run for update")
				}
				r.apiObj = new
				r.copied = true
			} else {
				return errors.Wrap(err,
					fmt.Sprintf("Failed to update status [%s]", r.String()))
			}
		}
	}
	r.apiObj = result
	return nil
}

func (r *pipelineRun) ensureCopy() {
	if !r.copied {
		r.apiObj = r.apiObj.DeepCopy()
		r.copied = true
	}
}
