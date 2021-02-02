package k8s

import (
	"fmt"
	"net/url"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	utils "github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	klog "k8s.io/klog/v2"
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
	InitState() error
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

// InitState initializes the state as 'new' if state was undefined (empty) before.
// The state's start time will be set to the object's creation time.
// Does not do anything if state is not undefined.
func (r *pipelineRun) InitState() error {
	r.ensureCopy()
	klog.V(3).Infof("Init State [%s]", r.String())
	return r.changeStatusAndUpdateSafely(func() error {

		if r.apiObj.Status.State != api.StateUndefined {
			return nil
		}

		newStateDetails := api.StateItem{
			State:     api.StateNew,
			StartedAt: r.apiObj.ObjectMeta.CreationTimestamp,
		}
		r.apiObj.Status.StateDetails = newStateDetails
		r.apiObj.Status.State = api.StateNew
		return nil
	})
}

// UpdateState set end time of current (defined) state (A) and store it to the history.
// if no current state is defined a new state (A) with cretiontime of the pipelinerun as start time is created.
// It also creates a new current state (B) with start time.
// Returns the state details of state A
func (r *pipelineRun) UpdateState(state api.State) (*api.StateItem, error) {
	if r.apiObj.Status.State == api.StateUndefined {
		return nil, fmt.Errorf("Cannot update uninitialize state")
	}
	r.ensureCopy()
	klog.V(3).Infof("Update State to %s [%s]", state, r.String())
	now := metav1.Now()
	oldStateDetails := r.apiObj.Status.StateDetails

	err := r.changeStatusAndUpdateSafely(func() error {

		currentStateDetails := r.apiObj.Status.StateDetails
		if currentStateDetails.State != oldStateDetails.State {
			return fmt.Errorf("State cannot be updated as it was changed concurrently from %q to %q", oldStateDetails.State, currentStateDetails.State)
		}
		if state == api.StatePreparing {
			r.apiObj.Status.StartedAt = &now
		}
		currentStateDetails.FinishedAt = now
		his := r.apiObj.Status.StateHistory
		his = append(his, currentStateDetails)

		newStateDetails := api.StateItem{State: state, StartedAt: now}
		if state == api.StateFinished {
			newStateDetails.FinishedAt = now
		}

		r.apiObj.Status.StateDetails = newStateDetails
		r.apiObj.Status.StateHistory = his
		r.apiObj.Status.State = state
		return nil
	})

	if err != nil {
		return nil, err
	}
	his := r.apiObj.Status.StateHistory
	hisLen := len(his)
	return &his[hisLen-1], nil
}

// String returns the full qualified name of the pipeline run
func (r *pipelineRun) String() string {
	return fmt.Sprintf("PipelineRun{name: %s, namespace: %s, state: %s}", r.GetName(), r.GetNamespace(), string(r.GetStatus().State))
}

// UpdateResult of the pipeline run
func (r *pipelineRun) UpdateResult(result api.Result) error {
	r.ensureCopy()
	return r.changeStatusAndUpdateSafely(func() error {
		r.apiObj.Status.Result = result
		now := metav1.Now()
		r.apiObj.Status.FinishedAt = &now
		return nil
	})
}

// UpdateContainer ...
func (r *pipelineRun) UpdateContainer(c *corev1.ContainerState) error {
	if c == nil {
		return nil
	}
	r.ensureCopy()
	return r.changeStatusAndUpdateSafely(func() error {
		r.apiObj.Status.Container = *c
		return nil
	})
}

// StoreErrorAsMessage stores the error as message in the status
func (r *pipelineRun) StoreErrorAsMessage(err error, message string) error {
	if err != nil {
		text := fmt.Sprintf("ERROR: %s [%s]: %s", utils.Trim(message), r.String(), err.Error())
		klog.V(3).Infof(text)
		return r.UpdateMessage(text)
	}
	return nil
}

// UpdateMessage stores string as message in the status
func (r *pipelineRun) UpdateMessage(message string) error {
	r.ensureCopy()

	return r.changeStatusAndUpdateSafely(func() error {
		old := r.apiObj.Status.Message
		if old != "" {
			his := r.apiObj.Status.History
			his = append(his, old)
			r.apiObj.Status.History = his
		}
		r.apiObj.Status.Message = utils.Trim(message)
		r.apiObj.Status.MessageShort = utils.ShortenMessage(message, 100)
		return nil
	})
}

// UpdateRunNamespace overrides the namespace in which the builds happens
func (r *pipelineRun) UpdateRunNamespace(ns string) error {
	r.ensureCopy()
	return r.changeStatusAndUpdateSafely(func() error {
		r.apiObj.Status.Namespace = ns
		return nil
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
		panic(fmt.Errorf("No factory provided to store updates [%s]", r.String()))
	}
	r.ensureCopy()
	start := time.Now()
	r.apiObj.ObjectMeta.Finalizers = finalizerList
	result, err := r.client.Update(r.apiObj)
	end := time.Now()
	elapsed := end.Sub(start)
	klog.V(3).Infof("finish update finalizer after %s in %s", elapsed, r.apiObj.Name)
	if err != nil {
		return errors.Wrap(err,
			fmt.Sprintf("Failed to update finalizers [%s]", r.String()))
	}
	r.apiObj = result
	return nil
}

// changeStatusAndUpdateSafely executes `change` and writes the
// status of the underlying PipelineRun object to storage afterwards.
// `change` is expected to mutate only the status of the underlying
// object, not more.
// In case of a conflict (object in storage is different version than
// ours), the update is retried with backoff:
//     - wait
//     - fetch object from storage
//     - run `change`
//     - write object status to storage
// After too many conflicts retrying is aborted, in which case an
// error is returned.
// Non-conflict errors are returned without retrying.
//
// Pitfall: If the underlying PipelineRun object was changed in memory
// compared to the version in storage _before calling this function_,
// that change _gets_ persisted in case there's _no_ update conflict, but
// gets _lost_ in case there _is_ an update conflict! This is hard to find
// by tests, as those typically do not encounter update conflicts.
func (r *pipelineRun) changeStatusAndUpdateSafely(change func() error) error {
	if r.client == nil {
		panic(fmt.Errorf("No factory provided to store updates [%s]", r.String()))
	}

	isRetry := false
	var changeError error = nil
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var err error

		if isRetry {
			new, err := r.client.Get(r.apiObj.GetName(), metav1.GetOptions{})
			if err != nil {
				return errors.Wrap(err,
					"failed to fetch pipeline after update conflict")
			}
			r.apiObj = new
			r.copied = true
		} else {
			defer func() { isRetry = true }()
		}

		changeError = change()
		if changeError != nil {
			return nil
		}

		result, err := r.client.UpdateStatus(r.apiObj)
		if err == nil {
			r.apiObj = result
			return nil
		}
		return err
	})
	if changeError != nil {
		return changeError
	}

	return errors.Wrapf(err, "failed to update status [%s]", r.String())
}

func (r *pipelineRun) ensureCopy() {
	if !r.copied {
		r.apiObj = r.apiObj.DeepCopy()
		r.copied = true
	}
}
