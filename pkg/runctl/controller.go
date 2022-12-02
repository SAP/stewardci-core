/*
based on sample-controller from https://github.com/kubernetes/sample-controller/blob/7047ee6ceceef2118a2017bbfff4a86c1f56f1ca/controller.go
*/

package runctl

import (
	"context"
	"fmt"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/client/clientset/versioned/scheme"
	"github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"github.com/SAP/stewardci-core/pkg/maintenancemode"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/SAP/stewardci-core/pkg/runctl/metrics"
	run "github.com/SAP/stewardci-core/pkg/runctl/run"
	"github.com/SAP/stewardci-core/pkg/stewardlabels"
	"github.com/SAP/stewardci-core/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	klog "k8s.io/klog/v2"
)

const (
	// heartbeatStimulusKey is a special key inserted into the controller
	// work queue as heartbeat stimulus.
	// It is an invalid Kubernetes name to avoid conflicts with real
	// pipeline runs.
	heartbeatStimulusKey = "Heartbeat Stimulus"
)

var (
	// Interval for histogram creation set to prometheus default scrape interval
	meteringInterval = 1 * time.Minute
)

// Controller processes PipelineRun resources
type Controller struct {
	factory              k8s.ClientFactory
	pipelineRunFetcher   k8s.PipelineRunFetcher
	pipelineRunSynced    cache.InformerSynced
	tektonTaskRunsSynced cache.InformerSynced
	workqueue            workqueue.RateLimitingInterface
	testing              *controllerTesting
	recorder             record.EventRecorder
	pipelineRunLister    v1alpha1.PipelineRunLister
	pipelineRunStore     cache.Store

	heartbeatInterval time.Duration
	heartbeatLogLevel *klog.Level
}

type controllerTesting struct {
	createRunManagerStub       run.Manager
	newRunManagerStub          func(k8s.ClientFactory, secrets.SecretProvider) run.Manager
	loadPipelineRunsConfigStub func(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error)
	isMaintenanceModeStub      func(ctx context.Context) (bool, error)
}

// ControllerOpts stores options for the construction of a Controller
// instance.
type ControllerOpts struct {
	// HeartbeatInterval is the interval for heartbeats.
	// If zero or negative, heartbeats are disabled.
	HeartbeatInterval time.Duration

	// HeartbeatLogLevel is a pointer to a klog log level to be used for
	// logging heartbeats.
	// If nil, heartbeat logging is disabled and heartbeats are only
	// exposed via metric.
	HeartbeatLogLevel *klog.Level
}

// NewController creates new Controller
func NewController(factory k8s.ClientFactory, opts ControllerOpts) *Controller {
	pipelineRunInformer := factory.StewardInformerFactory().Steward().V1alpha1().PipelineRuns()
	pipelineRunLister := pipelineRunInformer.Lister()
	pipelineRunFetcher := k8s.NewListerBasedPipelineRunFetcher(pipelineRunInformer.Lister())
	tektonTaskRunInformer := factory.TektonInformerFactory().Tekton().V1beta1().TaskRuns()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.V(3).Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: factory.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "runController"})

	controller := &Controller{
		factory:            factory,
		pipelineRunFetcher: pipelineRunFetcher,
		pipelineRunLister:  pipelineRunLister,
		pipelineRunSynced:  pipelineRunInformer.Informer().HasSynced,

		tektonTaskRunsSynced: tektonTaskRunInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), metrics.WorkqueueName),
		recorder:             recorder,
		pipelineRunStore:     pipelineRunInformer.Informer().GetStore(),
	}

	controller.heartbeatInterval = opts.HeartbeatInterval
	if opts.HeartbeatLogLevel != nil {
		copyOfValue := *opts.HeartbeatLogLevel
		controller.heartbeatLogLevel = &copyOfValue
	}

	pipelineRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.addPipelineRun,
		UpdateFunc: func(old, new interface{}) {
			controller.addPipelineRun(new)
		},
	})
	tektonTaskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleTektonTaskRun,
		UpdateFunc: func(old, new interface{}) {
			controller.handleTektonTaskRun(new)
		},
	})

	return controller
}

// meterAllPipelineRunsPeriodic observes certain metrics of all existing pipeline runs (in the informer cache).
func (c *Controller) meterAllPipelineRunsPeriodic() {
	klog.V(4).Infof("metering all pipeline runs")
	objs := c.pipelineRunStore.List()
	for _, obj := range objs {
		pipelineRun := obj.(*api.PipelineRun)

		// do not meter delays caused by finalizers
		if pipelineRun.DeletionTimestamp.IsZero() {
			metrics.PipelineRunsPeriodic.Observe(pipelineRun)
		}
	}
}

// Run runs the controller
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(2).Infof("Sync cache")
	if ok := cache.WaitForCacheSync(stopCh, c.pipelineRunSynced, c.tektonTaskRunsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.V(2).Infof("Starting metering of pipeline runs with interval %v", meteringInterval)
	go wait.Until(c.meterAllPipelineRunsPeriodic, meteringInterval, stopCh)

	if c.heartbeatInterval > 0 {
		klog.V(2).Infof("Starting controller heartbeat stimulator with interval %s", c.heartbeatInterval)
		go wait.Until(c.heartbeatStimulus, c.heartbeatInterval, stopCh)
	} else {
		klog.V(2).Info("Controller heartbeat is disabled")
	}

	klog.V(2).Infof("Start workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	klog.V(2).Infof("Workers running")

	<-stopCh
	klog.V(2).Infof("Workers stopped")
	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.V(5).Infof("Finished syncing '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) heartbeatStimulus() {
	c.workqueue.Add(heartbeatStimulusKey)
}

func (c *Controller) heartbeat() {
	if c.heartbeatLogLevel != nil {
		klog.V(*c.heartbeatLogLevel).InfoS("heartbeat")
	}
	metrics.ControllerHeartbeats.Inc()
}

func (c *Controller) changeState(pipelineRun k8s.PipelineRun, state api.State, ts metav1.Time) error {
	err := pipelineRun.UpdateState(state, ts)
	if err != nil {
		klog.V(3).Infof("Failed to UpdateState of [%s] to %q: %q", pipelineRun.String(), state, err.Error())
		return err
	}

	return nil
}

func (c *Controller) createRunManager(pipelineRun k8s.PipelineRun) run.Manager {
	if c.testing != nil && c.testing.createRunManagerStub != nil {
		return c.testing.createRunManagerStub
	}
	tenant := k8s.NewTenantNamespace(c.factory, pipelineRun.GetNamespace())
	workFactory := tenant.TargetClientFactory()
	return c.newRunManager(workFactory, tenant.GetSecretProvider())
}

func (c *Controller) newRunManager(workFactory k8s.ClientFactory, secretProvider secrets.SecretProvider) run.Manager {
	if c.testing != nil && c.testing.newRunManagerStub != nil {
		return c.testing.newRunManagerStub(workFactory, secretProvider)

	}
	return newRunManager(workFactory, secretProvider)
}

func (c *Controller) loadPipelineRunsConfig(ctx context.Context) (*cfg.PipelineRunsConfigStruct, error) {
	if c.testing != nil && c.testing.loadPipelineRunsConfigStub != nil {
		return c.testing.loadPipelineRunsConfigStub(ctx)
	}
	return cfg.LoadPipelineRunsConfig(ctx, c.factory)
}

func (c *Controller) isMaintenanceMode(ctx context.Context) (bool, error) {
	if c.testing != nil && c.testing.isMaintenanceModeStub != nil {
		return c.testing.isMaintenanceModeStub(ctx)
	}
	return maintenancemode.IsMaintenanceMode(ctx, c.factory)
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {

	if key == heartbeatStimulusKey {
		c.heartbeat()
		return nil
	}

	ctx := context.Background()

	// Initial checks on cached pipelineRun
	pipelineRunAPIObj, err := c.pipelineRunFetcher.ByKey(ctx, key)
	if err != nil {
		return err
	}
	// If pipelineRun is not found there is nothing to sync
	if pipelineRunAPIObj == nil {
		return nil
	}
	// don't process if labelled as to be ignored
	if stewardlabels.IsLabelledAsIgnore(pipelineRunAPIObj) {
		return nil
	}
	// fast exit - no finalizer cleanup needed
	if pipelineRunAPIObj.Status.State == api.StateFinished && !utils.StringSliceContains(pipelineRunAPIObj.ObjectMeta.Finalizers, k8s.FinalizerName) {
		return nil
	}

	// Get real pipelineRun bypassing cache
	pipelineRun, err := k8s.NewPipelineRun(ctx, pipelineRunAPIObj, c.factory)
	if err != nil {
		return err
	}

	// If pipelineRun is not found there is nothing to sync
	if pipelineRun == nil {
		return nil
	}

	// fast exit with finalizer cleanup
	if pipelineRun.GetStatus().State == api.StateFinished {
		return pipelineRun.DeleteFinalizerIfExists(ctx)
	}

	// Check if object has deletion timestamp ...
	if pipelineRun.HasDeletionTimestamp() {
		runManager := c.createRunManager(pipelineRun)
		err = runManager.Cleanup(ctx, pipelineRun)
		if err != nil {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonCleaningFailed, err.Error())
			return err
		}
		return c.updateStateAndResult(ctx, pipelineRun, api.StateFinished, api.ResultDeleted, metav1.Now())
	}
	// ... if not, try to add finalizer if missing
	pipelineRun.AddFinalizer(ctx)

	// Check if pipeline run is aborted
	if err := c.handleAborted(ctx, pipelineRun); err != nil {
		return err
	}

	// As soon as we have a result we can cleanup
	if pipelineRun.GetStatus().Result != api.ResultUndefined && pipelineRun.GetStatus().State != api.StateCleaning {
		err = c.changeState(pipelineRun, api.StateCleaning, metav1.Now())
		if err != nil {
			klog.V(1).Infof("WARN: change state to cleaning failed with: %s", err.Error())
		}
	}

	// Init state when undefined
	if pipelineRun.GetStatus().State == api.StateUndefined {

		err = pipelineRun.InitState()
		if err != nil {
			return err
		}
	}

	if pipelineRun.GetStatus().State == api.StateNew {
		maintenanceMode, err := c.isMaintenanceMode(ctx)
		if err != nil {
			return err
		}
		if maintenanceMode {
			err := fmt.Errorf("pipeline execution is paused while the system is in maintenance mode")
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeNormal, api.EventReasonMaintenanceMode, err.Error())
			// Return error that the pipeline stays in the queue and will be processed after switching back to normal mode.
			return err
		}
		if err = c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StatePreparing, metav1.Now()); err != nil {
			return err
		}
		metrics.PipelineRunsStarted.Inc()
	}

	runManager := c.createRunManager(pipelineRun)

	// Process pipeline run based on current state
	switch state := pipelineRun.GetStatus().State; state {
	case api.StatePreparing:
		// the configuration should be loaded once per sync to avoid inconsistencies
		// in case of concurrent configuration changes
		pipelineRunsConfig, err := c.loadPipelineRunsConfig(ctx)
		if err != nil {
			return c.onGetRunError(ctx, pipelineRunAPIObj, pipelineRun, err, api.StateFinished, api.ResultErrorInfra, "failed to load configuration for pipeline runs")
		}
		namespace, auxNamespace, err := runManager.Prepare(ctx, pipelineRun, pipelineRunsConfig)
		if err != nil {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonPreparingFailed, err.Error())
			resultClass := serrors.GetClass(err)
			// In case we have a result we can cleanup. Otherwise we retry in the next iteration.
			if resultClass != api.ResultUndefined {
				pipelineRun.UpdateMessage(err.Error())
				pipelineRun.StoreErrorAsMessage(err, "preparing failed")
				return c.updateStateAndResult(ctx, pipelineRun, api.StateCleaning, resultClass, metav1.Now())
			}
			return err
		}

		pipelineRun.UpdateRunNamespace(namespace)
		pipelineRun.UpdateAuxNamespace(auxNamespace)

		if err = c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StateWaiting, metav1.Now()); err != nil {
			return err
		}

		// TODO: Move Start to StateWaiting and do proper commit
		if err = runManager.Start(ctx, pipelineRun, pipelineRunsConfig); err != nil {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonPreparingFailed, err.Error())
			resultClass := serrors.GetClass(err)
			// In case we have a result we can cleanup. Otherwise we retry in the next iteration.
			if resultClass != api.ResultUndefined {
				pipelineRun.UpdateMessage(err.Error())
				pipelineRun.StoreErrorAsMessage(err, "preparing failed")
				return c.updateStateAndResult(ctx, pipelineRun, api.StateCleaning, resultClass, metav1.Now())
			}
			return err
		}

	case api.StateWaiting:
		run, err := runManager.GetRun(ctx, pipelineRun)
		if err != nil {
			return c.onGetRunError(ctx, pipelineRunAPIObj, pipelineRun, err, api.StateCleaning, api.ResultErrorInfra, "waiting failed")
		}
		started := run.GetStartTime()
		if started != nil {
			if err := c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StateRunning, *started); err != nil {
				return err
			}
		}
	case api.StateRunning:
		run, err := runManager.GetRun(ctx, pipelineRun)
		if err != nil {
			return c.onGetRunError(ctx, pipelineRunAPIObj, pipelineRun, err, api.StateCleaning, api.ResultErrorInfra, "running failed")
		}
		containerInfo := run.GetContainerInfo()
		pipelineRun.UpdateContainer(containerInfo)
		if finished, result := run.IsFinished(); finished {
			pipelineRun.UpdateMessage(run.GetMessage())
			return c.updateStateAndResult(ctx, pipelineRun, api.StateCleaning, result, *run.GetCompletionTime())
		}
		// commit container update
		err = c.commitStatusAndMeter(ctx, pipelineRun)
		if err != nil {
			return err
		}

	case api.StateCleaning:
		err = runManager.Cleanup(ctx, pipelineRun)
		if err != nil {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonCleaningFailed, err.Error())
		}
		if err := c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StateFinished, metav1.Now()); err != nil {
			return err
		}
		return pipelineRun.DeleteFinalizerIfExists(ctx)
	default:
		klog.V(2).Infof("Skip PipelineRun with state %s", pipelineRun.GetStatus().State)
	}
	return nil
}

func (c *Controller) onGetRunError(ctx context.Context, pipelineRunAPIObj *api.PipelineRun, pipelineRun k8s.PipelineRun, err error, state api.State, result api.Result, message string) error {
	c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonRunningFailed, err.Error())
	if serrors.IsRecoverable(err) {
		return err
	}
	pipelineRun.StoreErrorAsMessage(err, message)
	return c.updateStateAndResult(ctx, pipelineRun, state, result, metav1.Now())
}

func (c *Controller) changeAndCommitStateAndMeter(ctx context.Context, pipelineRun k8s.PipelineRun, state api.State, ts metav1.Time) error {
	if err := c.changeState(pipelineRun, state, ts); err != nil {
		return err
	}
	return c.commitStatusAndMeter(ctx, pipelineRun)
}

func (c *Controller) updateStateAndResult(ctx context.Context, pipelineRun k8s.PipelineRun, state api.State, result api.Result, ts metav1.Time) error {
	pipelineRun.UpdateResult(result, ts)
	if err := c.changeAndCommitStateAndMeter(ctx, pipelineRun, state, ts); err != nil {
		return err
	}
	metrics.PipelineRunsResult.Observe(pipelineRun.GetStatus().Result)
	if state == api.StateFinished {
		return pipelineRun.DeleteFinalizerIfExists(ctx)
	}
	return nil
}

func (c *Controller) commitStatusAndMeter(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	start := time.Now()
	finishedStates, err := pipelineRun.CommitStatus(ctx)
	if err != nil {
		klog.V(6).Infof("commitStatus failed with error %s", err.Error())
		return err
	}
	end := time.Now()
	elapsed := end.Sub(start)
	klog.V(6).Infof("commit of %q took %v", pipelineRun.String(), elapsed)
	metrics.UpdatesLatency.Observe("UpdateState", elapsed)
	for _, finishedState := range finishedStates {
		metrics.PipelineRunsStateFinished.Observe(finishedState)
	}
	return nil
}

// handleAborted checks if pipeline run should be aborted.
// If the user requested abortion it updates message, result and state
// to trigger a cleanup.
func (c *Controller) handleAborted(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	intent := pipelineRun.GetSpec().Intent
	if intent == api.IntentAbort && pipelineRun.GetStatus().Result == api.ResultUndefined {
		pipelineRun.UpdateMessage("Aborted")
		return c.updateStateAndResult(ctx, pipelineRun, api.StateCleaning, api.ResultAborted, metav1.Now())
	}
	return nil
}

func (c *Controller) addPipelineRun(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.V(4).Infof("Add to workqueue '%s'", key)
	c.workqueue.Add(key)
}

// handleTektonTaskRun takes any resource implementing metav1.Object and attempts
// to find the PipelineRun resource that 'owns' it. It does this by looking for
// a specific annotation. If such annotation exists, the named PipelineRun
// is put into the controller's work queue to be processed.
func (c *Controller) handleTektonTaskRun(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(3).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetSelfLink())
	annotations := object.GetAnnotations()
	runKey := annotations[annotationPipelineRunKey]
	if runKey != "" {
		klog.V(4).Infof("Add to workqueue '%s'", runKey)
		c.workqueue.Add(runKey)
	}
}
