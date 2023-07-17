package k8s

import (
	"context"
	"fmt"
	"net/url"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/client/clientset/versioned/scheme"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/metrics"
	utils "github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	ref "k8s.io/client-go/tools/reference"
	"k8s.io/client-go/util/retry"
	klog "k8s.io/klog/v2"
)

// PipelineRun is a set of utility functions working on an underlying
// api.PipelineRun API object.
type PipelineRun interface {
	fmt.Stringer

	// GetReference returns a reference to the original PipelineRun API object
	// this instance has been created with.
	GetReference() *v1.ObjectReference

	// GetAPIObject returns the underlying PipelineRun API object which
	// may contain uncommitted changes.
	// The underlying PipelineRun API object MUST NOT be modified.
	GetAPIObject() *api.PipelineRun

	// GetStatus returns the status of the underlying PipelineRun API object.
	// The status of the underlying PipelineRun object MUST NOT be modified
	// directly. Instead the provided update functions must be used.
	GetStatus() *api.PipelineStatus

	// GetSpec returns the spec of the underlying PipelineRun API object.
	// The spec of the underlying PipelineRun object MUST NOT be modified.
	GetSpec() *api.PipelineSpec

	// GetName returns the name of the underlying PipelineRun API object.
	GetName() string

	// GetKey returns the key of the pipeline run to be used in the
	// controller workqueue or informer cache.
	GetKey() string

	// GetRunNamespace returns the namespace in which the build takes place
	// if available in the status of the underlying PipelineRun API object,
	// otherwise the empty string.
	GetRunNamespace() string

	// GetAuxNamespace returns the namespace hosting auxiliary services
	// if available in the status of the underlying PipelineRun API object,
	// otherwise the empty string.
	GetAuxNamespace() string

	// GetNamespace returns the namespace of the underlying pipelineRun object
	GetNamespace() string

	// GetValidatedJenkinsfileRepoServerURL validates the Jenkinsfile
	// Git repository URL and returns the URL without path.
	GetValidatedJenkinsfileRepoServerURL() (string, error)

	// HasDeletionTimestamp return whether the underlying PipelineRun API
	// object has a deletion timestamp set.
	HasDeletionTimestamp() bool

	// AddFinalizerAndCommitIfNotPresent adds the Steward finalizer to the list
	// of finalizers of the underlying PipelineRun API object if it is not
	// present already. The change is immediately committed.
	//
	// There must not be any other pending changes.
	AddFinalizerAndCommitIfNotPresent(ctx context.Context) error

	// CommitStatus writes the status of the underlying PipelineRun object to
	// storage.
	//
	// In case of a conflict (object in storage is different version than
	// ours), the update is retried with backoff:
	//   - wait
	//   - fetch object from storage
	//   - re-apply recorded status changes
	//   - update object status in storage
	//
	// After too many conflicts retrying is aborted, in which case an
	// error is returned.
	//
	// Non-conflict errors are returned without retrying.
	//
	// Pitfall: If the underlying PipelineRun API object was changed in memory
	// compared to the version in storage _before calling this function_,
	// that change _gets_ persisted in case there's _no_ update conflict, but
	// gets _lost_ in case there _is_ an update conflict! This is hard to find
	// by tests, as those typically do not encounter update conflicts.
	CommitStatus(ctx context.Context) ([]*api.StateItem, error)

	// DeleteFinalizerAndCommitIfExists deletes the Steward finalizer from the
	// list of finalizers of the underlying PipelineRun API object if it is
	// present. The change is immediately committed.
	//
	// There must not be any other pending changes.
	DeleteFinalizerAndCommitIfExists(ctx context.Context) error

	// InitState initializes the state as 'new' if it was undefined (empty)
	// before.
	// The state's start time will be set to the object's creation time.
	// Fails if a state is set already.
	InitState(ctx context.Context) error

	// UpdateState sets timestamp as end time of current (defined) state (A) and
	// stores it in the history. If no current state is defined a new state (A)
	// with creation time of the pipeline run as start time is created. It also
	// creates a new current state (B) with timestamp as start time.
	UpdateState(ctx context.Context, state api.State, timestamp metav1.Time) error

	// UpdateResult updates the result and finish timestamp.
	UpdateResult(ctx context.Context, result api.Result, finishedAt metav1.Time)

	// UpdateContainer updates the container info in the status.
	UpdateContainer(ctx context.Context, newContainerState *corev1.ContainerState)

	// StoreErrorAsMessage stores err with prefix as message in the status.
	// If err is nil, the message is NOT updated.
	StoreErrorAsMessage(ctx context.Context, err error, prefix string) error

	// UpdateRunNamespace sets namespace as the run namespace in the status.
	UpdateRunNamespace(namespace string)

	// UpdateAuxNamespace sets namespace as the auxiliary namespace in the
	// status.
	UpdateAuxNamespace(namespace string)

	// UpdateMessage sets msg as message in the status.
	UpdateMessage(msg string)
}

// pipelineRun is the (only) implementation of interface PipelineRun.
type pipelineRun struct {
	client          stewardv1alpha1.PipelineRunInterface
	reference       *v1.ObjectReference
	apiObj          *api.PipelineRun
	copied          bool
	changes         []changeFunc
	commitRecorders []commitRecorderFunc
}

type changeFunc func(*api.PipelineStatus) (commitRecorderFunc, error)

type commitRecorderFunc func() *api.StateItem

// NewPipelineRun creates a new instance of PipelineRun based on the given apiObj.
//
// If a factory is provided a new version of the pipelinerun is fetched.
// All changes are done on the fetched object.
// If no pipeline run can be found matching the apiObj, nil,nil is returned.
// An error is only returned if a Get for the pipelinerun returns an error other than a NotFound error.
// If you call with factory nil you can only use the Get* functions
// If you use functions changing the pipeline run without factroy set you will get an error.
// The provided PipelineRun object is never modified and copied as late as possible.
func NewPipelineRun(ctx context.Context, apiObj *api.PipelineRun, factory ClientFactory) (PipelineRun, error) {
	if apiObj == nil {
		return nil, nil
	}

	reference, err := ref.GetReference(scheme.Scheme, apiObj)
	if err != nil {
		return nil, fmt.Errorf("cannot create reference for apiObj: %v", apiObj)
	}

	if factory == nil {
		return &pipelineRun{
			reference: reference,
			apiObj:    apiObj,
			copied:    false,
		}, nil
	}
	client := factory.StewardV1alpha1().PipelineRuns(apiObj.GetNamespace())
	fetchedAPIObj, err := client.Get(ctx, apiObj.GetName(), metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &pipelineRun{
		reference:       reference,
		apiObj:          fetchedAPIObj,
		copied:          true,
		client:          client,
		changes:         []changeFunc{},
		commitRecorders: []commitRecorderFunc{},
	}, nil
}

// GetReference implements part of interface `PipelineRun`.
func (r *pipelineRun) GetReference() *v1.ObjectReference {
	return r.reference
}

// GetAPIObject implements part of interface `PipelineRun`.
func (r *pipelineRun) GetAPIObject() *api.PipelineRun {
	return r.apiObj
}

// GetRunNamespace implements part of interface `PipelineRun`.
func (r *pipelineRun) GetRunNamespace() string {
	return r.apiObj.Status.Namespace
}

// GetAuxNamespace implements part of interface `PipelineRun`.
func (r *pipelineRun) GetAuxNamespace() string {
	return r.apiObj.Status.AuxiliaryNamespace
}

// GetKey implements part of interface `PipelineRun`.
func (r *pipelineRun) GetKey() string {
	key, _ := cache.MetaNamespaceKeyFunc(r.apiObj)
	return key
}

// GetNamespace implements part of interface `PipelineRun`.
func (r *pipelineRun) GetNamespace() string {
	return r.apiObj.GetNamespace()
}

// GetValidatedJenkinsfileRepoServerURL implements part of interface `PipelineRun`.
func (r *pipelineRun) GetValidatedJenkinsfileRepoServerURL() (string, error) {
	urlString := r.GetSpec().JenkinsFile.URL
	repoURL, err := url.Parse(urlString)
	if err != nil {
		return "", errors.Wrapf(err, "value %q of field spec.jenkinsFile.url is invalid [%s]", urlString, r.String())
	}
	if repoURL.Scheme != "http" && repoURL.Scheme != "https" {
		return "", fmt.Errorf("value %q of field spec.jenkinsFile.url is invalid [%s]: scheme not supported: %q", urlString, r.String(), repoURL.Scheme)
	}
	return fmt.Sprintf("%s://%s", repoURL.Scheme, repoURL.Host), nil
}

// GetName implements part of interface `PipelineRun`.
func (r *pipelineRun) GetName() string {
	return r.apiObj.GetName()
}

// GetStatus implements part of interface `PipelineRun`.
func (r *pipelineRun) GetStatus() *api.PipelineStatus {
	return &r.apiObj.Status
}

// GetSpec implements part of interface `PipelineRun`.
func (r *pipelineRun) GetSpec() *api.PipelineSpec {
	return &r.apiObj.Spec
}

// InitState implements part of interface `PipelineRun`.
func (r *pipelineRun) InitState(ctx context.Context) error {
	logger := klog.FromContext(ctx)
	r.ensureCopy()
	logger.V(3).Info("Set state to 'new'")
	return r.changeStatusAndStoreForRetry(func(s *api.PipelineStatus) (commitRecorderFunc, error) {

		if s.State != api.StateUndefined {
			return nil, fmt.Errorf("cannot initialize multiple times")
		}

		newStateDetails := api.StateItem{
			State:     api.StateNew,
			StartedAt: r.apiObj.ObjectMeta.CreationTimestamp,
		}
		s.StateDetails = newStateDetails
		s.State = api.StateNew
		return nil, nil
	})
}

// UpdateState implements part of interface `PipelineRun`.
func (r *pipelineRun) UpdateState(ctx context.Context, state api.State, timestamp metav1.Time) error {
	if r.apiObj.Status.State == api.StateUndefined {
		if err := r.InitState(ctx); err != nil {
			return err
		}
	}
	r.ensureCopy()
	oldStateDetails := r.apiObj.Status.StateDetails

	return r.changeStatusAndStoreForRetry(func(s *api.PipelineStatus) (commitRecorderFunc, error) {
		currentStateDetails := s.StateDetails
		if currentStateDetails.State != oldStateDetails.State {
			return nil, fmt.Errorf("state cannot be updated as it was changed concurrently from %q to %q", oldStateDetails.State, currentStateDetails.State)
		}
		if state == api.StatePreparing {
			s.StartedAt = &timestamp
		}
		currentStateDetails.FinishedAt = timestamp
		his := s.StateHistory
		his = append(his, currentStateDetails)

		commitRecorderFunc := func() *api.StateItem {
			return &currentStateDetails
		}
		newStateDetails := api.StateItem{State: state, StartedAt: timestamp}
		if state == api.StateFinished {
			newStateDetails.FinishedAt = timestamp
		}

		s.StateDetails = newStateDetails
		s.StateHistory = his
		s.State = state
		return commitRecorderFunc, nil
	})
}

// String implements interface fmt.Stringer.
func (r *pipelineRun) String() string {
	return fmt.Sprintf("PipelineRun{name: %s, namespace: %s, state: %s}", r.GetName(), r.GetNamespace(), string(r.GetStatus().State))
}

// UpdateResult implements part of interface `PipelineRun`.
func (r *pipelineRun) UpdateResult(ctx context.Context, result api.Result, finishedAT metav1.Time) {
	r.ensureCopy()
	r.mustChangeStatusAndStoreForRetry(func(s *api.PipelineStatus) (commitRecorderFunc, error) {
		s.Result = result
		s.FinishedAt = &finishedAT
		return nil, nil
	})
}

// UpdateContainer implements part of interface `PipelineRun`.
func (r *pipelineRun) UpdateContainer(ctx context.Context, newContainerState *corev1.ContainerState) {
	if newContainerState == nil {
		return
	}
	r.ensureCopy()
	r.mustChangeStatusAndStoreForRetry(func(s *api.PipelineStatus) (commitRecorderFunc, error) {
		s.Container = *newContainerState
		return nil, nil
	})
}

// StoreErrorAsMessage implements part of interface `PipelineRun`.
func (r *pipelineRun) StoreErrorAsMessage(ctx context.Context, err error, prefix string) error {
	if err != nil {
		text := fmt.Sprintf("ERROR: %s [%s]: %s", utils.Trim(prefix), r.String(), err.Error())
		r.UpdateMessage(text)
	}
	return nil
}

// UpdateMessage implements part of interface `PipelineRun`.
func (r *pipelineRun) UpdateMessage(msg string) {
	r.ensureCopy()

	r.mustChangeStatusAndStoreForRetry(func(s *api.PipelineStatus) (commitRecorderFunc, error) {
		old := s.Message
		if old != "" {
			his := s.History
			his = append(his, old)
			s.History = his
		}
		s.Message = utils.Trim(msg)
		s.MessageShort = utils.ShortenMessage(msg, 100)
		return nil, nil
	})
}

// UpdateRunNamespace implements part of interface `PipelineRun`.
func (r *pipelineRun) UpdateRunNamespace(ns string) {
	r.ensureCopy()
	r.mustChangeStatusAndStoreForRetry(func(s *api.PipelineStatus) (commitRecorderFunc, error) {
		s.Namespace = ns
		return nil, nil
	})
}

// UpdateAuxNamespace implements part of interface `PipelineRun`.
func (r *pipelineRun) UpdateAuxNamespace(ns string) {
	r.ensureCopy()
	r.mustChangeStatusAndStoreForRetry(func(s *api.PipelineStatus) (commitRecorderFunc, error) {
		s.AuxiliaryNamespace = ns
		return nil, nil
	})
}

// HasDeletionTimestamp implements part of interface `PipelineRun`.
func (r *pipelineRun) HasDeletionTimestamp() bool {
	return !r.apiObj.ObjectMeta.DeletionTimestamp.IsZero()
}

// AddFinalizerAndCommitIfNotPresent implements part of interface `PipelineRun`.
func (r *pipelineRun) AddFinalizerAndCommitIfNotPresent(ctx context.Context) error {
	r.mustBeChangeable()
	r.mustNotHavePendingChanges()

	changed, finalizerList := utils.AddStringIfMissing(r.apiObj.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		err := r.commitFinalizerListExclusively(ctx, finalizerList)
		return err
	}
	return nil
}

// Panics if the instance holds any uncommitted changes.
func (r *pipelineRun) DeleteFinalizerAndCommitIfExists(ctx context.Context) error {
	r.mustBeChangeable()
	r.mustNotHavePendingChanges()

	changed, finalizerList := utils.RemoveString(r.apiObj.ObjectMeta.Finalizers, FinalizerName)
	if changed {
		return r.commitFinalizerListExclusively(ctx, finalizerList)
	}
	return nil
}

func (r *pipelineRun) commitFinalizerListExclusively(ctx context.Context, finalizerList []string) error {
	logger := klog.FromContext(ctx)

	r.mustBeChangeable()
	r.mustNotHavePendingChanges()

	r.ensureCopy()
	start := time.Now()
	r.apiObj.ObjectMeta.Finalizers = finalizerList
	result, err := r.client.Update(ctx, r.apiObj, metav1.UpdateOptions{})
	elapsed := time.Since(start)
	logger.V(4).Info("Updated finalizers", "duration", elapsed)
	if err != nil {
		return errors.Wrap(err,
			fmt.Sprintf("failed to update finalizers [%s]", r.String()))
	}
	r.apiObj = result
	return nil
}

// mustChangeStatusAndStoreForRetry calls changeStatusAndStoreForRetry and
// panics in case of an error.
func (r *pipelineRun) mustChangeStatusAndStoreForRetry(change changeFunc) {
	err := r.changeStatusAndStoreForRetry((change))
	if err != nil {
		panic(err)
	}
}

// changeStatusAndStoreForRetry receives a function applying changes to pipelinerun.Status
// This function get executed on the current memory representation of the pipeline run
// and remembered so that it can be re-applied later in case of a re-try. The change function
// must only apply changes to pipelinerun.Status.
func (r *pipelineRun) changeStatusAndStoreForRetry(change changeFunc) error {
	commitRecorder, err := change(r.GetStatus())
	if err == nil {
		r.changes = append(r.changes, change)
		r.commitRecorders = append(r.commitRecorders, commitRecorder)
	}

	return err
}

// CommitStatus implements part of interface `PipelineRun`.
func (r *pipelineRun) CommitStatus(ctx context.Context) ([]*api.StateItem, error) {
	logger := klog.FromContext(ctx)

	r.mustBeChangeable()

	logger.V(5).Info("Committing pipeline run status")
	if len(r.changes) == 0 {
		logger.V(5).Info("There is no status change to commit")
		return nil, nil
	}

	retryCount := uint64(0)
	defer func(start time.Time) {
		if retryCount > 0 {
			codeLocationSkipFrames := uint16(1)
			codeLocation := metrics.CodeLocation(codeLocationSkipFrames)
			latency := time.Since(start)
			metrics.Retries.Observe(codeLocation, retryCount, latency)
			logger.V(5).Info("Retry was required",
				"location", codeLocation,
				"count", retryCount,
				"latency", latency,
			)
		}
	}(time.Now())

	var changeError error
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var err error

		if retryCount > 0 {
			logger.V(5).Info("Reloading pipeline run after status commit failed due to conflict")
			fetchedAPIObj, err := r.client.Get(ctx, r.apiObj.GetName(), metav1.GetOptions{})
			if err != nil {
				return errors.Wrap(err,
					"failed to fetch pipeline after update conflict")
			}

			logger.V(5).Info("Applying status changes again", "count", len(r.changes))
			changeError = r.redoChanges(ctx, fetchedAPIObj)
			if changeError != nil {
				return nil
			}
		}

		result, err := r.client.UpdateStatus(ctx, r.apiObj, metav1.UpdateOptions{})
		if err == nil {
			r.apiObj = result
			return nil
		}
		retryCount++
		return err
	})
	r.changes = []changeFunc{}
	if changeError != nil {
		return nil, changeError
	}

	return r.getNonEmptyStateItems(), errors.Wrapf(err, "failed to update status [%s]", r.String())
}

func (r *pipelineRun) getNonEmptyStateItems() []*api.StateItem {
	result := []*api.StateItem{}
	for _, recorder := range r.commitRecorders {
		if recorder != nil {
			result = append(result, recorder())
		}
	}
	return result
}

func (r *pipelineRun) redoChanges(ctx context.Context, fetchedAPIObj *api.PipelineRun) error {
	logger := klog.FromContext(ctx)

	r.apiObj = fetchedAPIObj
	r.copied = true
	r.commitRecorders = []commitRecorderFunc{}
	for _, change := range r.changes {
		commitRecorder, err := change(r.GetStatus())
		if err != nil {
			logger.V(5).Error(err, "Failed to apply pipeline run status change")
			return err
		}
		r.commitRecorders = append(r.commitRecorders, commitRecorder)
	}
	return nil
}

func (r *pipelineRun) ensureCopy() {
	if !r.copied {
		r.apiObj = r.apiObj.DeepCopy()
		r.copied = true
	}
}

func (r *pipelineRun) mustBeChangeable() {
	if r.client == nil {
		panic(fmt.Errorf("read-only instance"))
	}
}

func (r *pipelineRun) mustNotHavePendingChanges() {
	if len(r.changes) > 0 {
		panic(fmt.Errorf("there are pending changes"))
	}
}
