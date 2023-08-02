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
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	k8ssecretprovider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/k8s"
	"github.com/SAP/stewardci-core/pkg/maintenancemode"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/SAP/stewardci-core/pkg/runctl/metrics"
	run "github.com/SAP/stewardci-core/pkg/runctl/run"
	"github.com/SAP/stewardci-core/pkg/runctl/runmgr"
	"github.com/SAP/stewardci-core/pkg/stewardlabels"
	"github.com/SAP/stewardci-core/pkg/utils"
	"github.com/go-logr/logr"
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
	// loggerName is the name of the run controller logger.
	loggerName = "runController"

	// reconcilerLoggerName is the name of the logger used in run controller's
	// reconciliation loop.
	reconcilerLoggerName = "reconciler"

	// heartbeatStimulusKey is a special key inserted into the controller
	// work queue as heartbeat stimulus.
	// It is an invalid Kubernetes name to avoid conflicts with real
	// pipeline runs.
	heartbeatStimulusKey = "Heartbeat Stimulus"

	errorMessageWaitingFailed   = "waiting failed"
	errorMessagePreparingFailed = "preparing failed"
	errorMessageRunningFailed   = "running failed"
)

var (
	// Interval for histogram creation set to prometheus default scrape interval
	meteringInterval = 1 * time.Minute

	defaultWaitTimeout = 10 * time.Minute
)

// Controller processes PipelineRun resources
type Controller struct {
	factory              k8s.ClientFactory
	pipelineRunFetcher   k8s.PipelineRunFetcher
	pipelineRunsSynced   cache.InformerSynced
	tektonTaskRunsSynced cache.InformerSynced
	workqueue            workqueue.RateLimitingInterface
	testing              *controllerTesting
	eventRecorder        record.EventRecorder
	pipelineRunStore     cache.Store

	heartbeatInterval time.Duration
	heartbeatLogLevel *klog.Level

	// logger *must* be initialized when creating Controller,
	// otherwise logging functions will access a nil sink and
	// panic.
	logger logr.Logger
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
func NewController(logger logr.Logger, factory k8s.ClientFactory, opts ControllerOpts) *Controller {
	logger = logger.WithName(loggerName)

	pipelineRunInformer := factory.StewardInformerFactory().Steward().V1alpha1().PipelineRuns()
	pipelineRunFetcher := k8s.NewListerBasedPipelineRunFetcher(pipelineRunInformer.Lister())
	tektonTaskRunInformer := factory.TektonInformerFactory().Tekton().V1beta1().TaskRuns()

	controller := &Controller{
		factory:            factory,
		pipelineRunFetcher: pipelineRunFetcher,
		pipelineRunsSynced: pipelineRunInformer.Informer().HasSynced,

		tektonTaskRunsSynced: tektonTaskRunInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), metrics.WorkqueueName),
		pipelineRunStore:     pipelineRunInformer.Informer().GetStore(),
		logger:               logger,
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: factory.CoreV1().Events("")})
	eventBroadcaster.StartEventWatcher(
		func(e *corev1.Event) {
			controller.logger.V(3).Info(
				"Event occurred",
				"object", klog.KRef(e.InvolvedObject.Namespace, e.InvolvedObject.Name),
				"fieldPath", e.InvolvedObject.FieldPath,
				"kind", e.InvolvedObject.Kind,
				"apiVersion", e.InvolvedObject.APIVersion,
				"type", e.Type,
				"reason", e.Reason,
				"message", e.Message,
			)
		},
	)
	controller.eventRecorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "runController"})

	controller.heartbeatInterval = opts.HeartbeatInterval
	if opts.HeartbeatLogLevel != nil {
		copyOfValue := *opts.HeartbeatLogLevel
		controller.heartbeatLogLevel = &copyOfValue
	}

	pipelineRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.addToWorkqueue,
		UpdateFunc: func(old, new interface{}) {
			controller.addToWorkqueue(new)
		},
	})
	tektonTaskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.addToWorkqueueFromAssociated,
		UpdateFunc: func(old, new interface{}) {
			controller.addToWorkqueueFromAssociated(new)
		},
	})

	return controller
}

// meterAllPipelineRunsPeriodic observes certain metrics of all existing pipeline runs (in the informer cache).
func (c *Controller) meterAllPipelineRunsPeriodic() {
	c.logger.V(4).Info("Metering all the pipeline runs")
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

	c.logger.V(2).Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.pipelineRunsSynced, c.tektonTaskRunsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	c.logger.V(2).Info("Starting periodic metering of pipeline runs", "interval", meteringInterval)
	go wait.Until(c.meterAllPipelineRunsPeriodic, meteringInterval, stopCh)

	if c.heartbeatInterval > 0 {
		c.logger.V(2).Info("Starting controller heartbeat stimulator", "interval", c.heartbeatInterval)
		go wait.Until(c.heartbeatStimulus, c.heartbeatInterval, stopCh)
	} else {
		c.logger.V(2).Info("Controller heartbeat stimulus is disabled")
	}

	c.logger.V(2).Info("Starting workers", "threadiness", threadiness)
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	c.logger.V(2).Info("Workers are running")

	<-stopCh
	c.logger.V(2).Info("Workers are stopped")
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
		c.logger.V(5).Info("Finished syncing", "key", key)
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
		c.logger.V(int(*c.heartbeatLogLevel)).Info("heartbeat")
	}
	metrics.ControllerHeartbeats.Inc()
}

func (c *Controller) changeState(ctx context.Context, pipelineRun k8s.PipelineRun, state api.State, ts metav1.Time) error {
	logger := klog.FromContext(ctx)

	logger.V(3).Info("Changing pipeline run state", "targetState", state)
	err := pipelineRun.UpdateState(ctx, state, ts)
	if err != nil {
		logger.V(3).Error(err, "Failed to change pipeline run state", "targetState", state)
		return err
	}

	return nil
}

func (c *Controller) createRunManager(pipelineRun k8s.PipelineRun) run.Manager {
	if c.testing != nil && c.testing.createRunManagerStub != nil {
		return c.testing.createRunManagerStub
	}
	namespace := pipelineRun.GetNamespace()
	secretsClient := c.factory.CoreV1().Secrets(namespace)
	secretProvider := k8ssecretprovider.NewProvider(secretsClient, namespace)
	return c.newRunManager(c.factory, secretProvider)
}

func (c *Controller) newRunManager(workFactory k8s.ClientFactory, secretProvider secrets.SecretProvider) run.Manager {
	if c.testing != nil && c.testing.newRunManagerStub != nil {
		return c.testing.newRunManagerStub(workFactory, secretProvider)

	}
	return runmgr.NewRunManager(workFactory, secretProvider)
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
	logger := c.logger.WithName(reconcilerLoggerName)

	if key == heartbeatStimulusKey {
		c.heartbeat()
		return nil
	}

	ctx := context.Background()

	pipelineRun, err := c.getPipelineRunToProcess(ctx, key)
	if pipelineRun == nil {
		// If pipelineRun is not found there is nothing to sync
		return err
	}

	ctx = klog.NewContext(ctx, logger)

	doReturn, err := c.handlePipelineRunFinalizerAndDeletion(ctx, pipelineRun)
	if doReturn || err != nil {
		return err
	}

	err = c.handlePipelineRunAbort(ctx, pipelineRun)
	if err != nil {
		return err
	}

	err = c.handlePipelineRunResultExistsButNotCleaned(ctx, pipelineRun)
	if err != nil {
		return err
	}

	doReturn, err = c.handlePipelineRunNew(ctx, pipelineRun)
	if doReturn || err != nil {
		return err
	}

	runManager := c.createRunManager(pipelineRun)

	// the configuration should be loaded once per sync to avoid inconsistencies
	// in case of concurrent configuration changes
	pipelineRunsConfig, doReturn, err := c.ensurePipelineRunsConfig(ctx, pipelineRun)
	if doReturn || err != nil {
		return err
	}

	doReturn, err = c.handlePipelineRunPrepare(ctx, runManager, pipelineRun, pipelineRunsConfig)
	if doReturn || err != nil {
		return err
	}

	doReturn, err = c.handlePipelineRunWaiting(ctx, runManager, pipelineRun, pipelineRunsConfig)
	if doReturn || err != nil {
		return err
	}

	doReturn, err = c.handlePipelineRunRunning(ctx, runManager, pipelineRun, pipelineRunsConfig)
	if doReturn || err != nil {
		return err
	}

	doReturn, err = c.handlePipelineRunCleaning(ctx, runManager, pipelineRun)
	if doReturn || err != nil {
		return err
	}

	return nil
}

func (c *Controller) getPipelineRunToProcess(ctx context.Context, key string) (k8s.PipelineRun, error) {
	// Get pipeline run from informer cache
	pipelineRunAPIObj, err := c.pipelineRunFetcher.ByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// Initial checks on cached pipelineRun
	if !c.isPipelineRunAPIObjThatNeedsProcessing(ctx, pipelineRunAPIObj) {
		return nil, nil
	}

	// Fetches latest revision from storage bypassing the informer cache
	pipelineRun, err := k8s.NewPipelineRun(ctx, pipelineRunAPIObj, c.factory)
	if err != nil {
		return nil, err
	}

	return pipelineRun, nil
}

// isPipelineRunAPIObjThatNeedsProcessing sorts out pipeline runs that do not
// need processing.
func (c *Controller) isPipelineRunAPIObjThatNeedsProcessing(ctx context.Context, pipelineRunAPIObj *api.PipelineRun) bool {
	// If pipelineRun is not found there is nothing to sync
	if pipelineRunAPIObj == nil {
		return false
	}
	// don't process if labelled as to be ignored
	if stewardlabels.IsLabelledAsIgnore(pipelineRunAPIObj) {
		return false
	}
	// fast exit - no finalizer cleanup needed
	if pipelineRunAPIObj.Status.State == api.StateFinished &&
		!utils.StringSliceContains(pipelineRunAPIObj.ObjectMeta.Finalizers, k8s.FinalizerName) {
		return false
	}
	return true
}

func (c *Controller) handlePipelineRunFinalizerAndDeletion(
	ctx context.Context,
	pipelineRun k8s.PipelineRun,
) (bool, error) {
	if pipelineRun.GetStatus().State == api.StateFinished {
		err := pipelineRun.DeleteFinalizerAndCommitIfExists(ctx)
		return true, err
	}

	if pipelineRun.HasDeletionTimestamp() {
		runManager := c.createRunManager(pipelineRun)
		err := runManager.Cleanup(ctx, pipelineRun)
		if err != nil {
			c.eventRecorder.Event(pipelineRun.GetReference(), corev1.EventTypeWarning, api.EventReasonCleaningFailed, err.Error())
			return true, err
		}
		return false, c.updateStateAndResult(ctx, pipelineRun, api.StateFinished, api.ResultDeleted, metav1.Now())
	}

	return false, pipelineRun.AddFinalizerAndCommitIfNotPresent(ctx)
}

func (c *Controller) handlePipelineRunResultExistsButNotCleaned(
	ctx context.Context,
	pipelineRun k8s.PipelineRun,
) error {
	ctx, _ = extendContextLoggerWithPipelineRunInfo(ctx, pipelineRun.GetAPIObject())

	result := pipelineRun.GetStatus().Result
	state := pipelineRun.GetStatus().State

	if result != api.ResultUndefined &&
		state != api.StateCleaning &&
		state != api.StateFinished {

		err := c.changeState(ctx, pipelineRun, api.StateCleaning, metav1.Now())
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func (c *Controller) handlePipelineRunNew(
	ctx context.Context,
	pipelineRun k8s.PipelineRun,
) (bool, error) {
	ctx, _ = extendContextLoggerWithPipelineRunInfo(ctx, pipelineRun.GetAPIObject())

	if pipelineRun.GetStatus().State == api.StateUndefined {
		if err := pipelineRun.InitState(ctx); err != nil {
			panic(err)
		}
	}

	if pipelineRun.GetStatus().State == api.StateNew {
		maintenanceMode, err := c.isMaintenanceMode(ctx)
		if err != nil {
			return true, err
		}
		if maintenanceMode {
			err := fmt.Errorf("pipeline execution is paused while the system is in maintenance mode")
			c.eventRecorder.Event(pipelineRun.GetReference(), corev1.EventTypeNormal, api.EventReasonMaintenanceMode, err.Error())
			// Return error that the pipeline stays in the queue and will be processed after switching back to normal mode.
			return true, err
		}
		if err = c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StatePreparing, metav1.Now()); err != nil {
			return true, err
		}
		metrics.PipelineRunsStarted.Inc()
	}
	return false, nil
}

func (c *Controller) ensurePipelineRunsConfig(ctx context.Context, pipelineRun k8s.PipelineRun) (*cfg.PipelineRunsConfigStruct, bool, error) {
	var pipelineRunsConfig *cfg.PipelineRunsConfigStruct

	ctx, logger := extendContextLoggerWithPipelineRunInfo(ctx, pipelineRun.GetAPIObject())

	state := pipelineRun.GetStatus().State
	// TODO do not assume in which phase the config is (not) needed
	if state == api.StatePreparing || state == api.StateWaiting {
		var err error
		pipelineRunsConfig, err = c.loadPipelineRunsConfig(ctx)
		if err != nil {
			var targetState api.State
			if state == api.StatePreparing {
				targetState = api.StateFinished
			} else {
				targetState = api.StateCleaning
			}
			err = c.onGetRunError(
				ctx,
				pipelineRun,
				err,
				targetState,
				api.ResultErrorInfra,
				"failed to load configuration for pipeline runs",
			)
			return nil, true, err
		}
		logger.V(3).Info("Loaded pipeline run config")
	}
	return pipelineRunsConfig, false, nil
}

func (c *Controller) handlePipelineRunPrepare(
	ctx context.Context,
	runManager run.Manager,
	pipelineRun k8s.PipelineRun,
	pipelineRunsConfig *cfg.PipelineRunsConfigStruct,
) (bool, error) {
	origCtx := ctx
	ctx, logger := extendContextLoggerWithPipelineRunInfo(origCtx, pipelineRun.GetAPIObject())

	if pipelineRun.GetStatus().State == api.StatePreparing {
		logger.V(3).Info("Preparing pipeline execution")

		namespace, auxNamespace, err := runManager.Prepare(ctx, pipelineRun, pipelineRunsConfig)
		if err != nil {
			c.eventRecorder.Event(pipelineRun.GetReference(), corev1.EventTypeWarning, api.EventReasonPreparingFailed, err.Error())
			resultClass := serrors.GetClass(err)
			// In case we have a result we can cleanup. Otherwise we retry in the next iteration.
			if resultClass != api.ResultUndefined {
				return true, c.handleResultError(ctx, pipelineRun, resultClass, errorMessagePreparingFailed, err)
			}
			return true, err
		}

		pipelineRun.UpdateRunNamespace(namespace)
		pipelineRun.UpdateAuxNamespace(auxNamespace)

		ctx, logger := extendContextLoggerWithPipelineRunInfo(origCtx, pipelineRun.GetAPIObject())
		logger.V(3).Info("Prepared pipeline execution")

		if err = c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StateWaiting, metav1.Now()); err != nil {
			return true, err
		}

		// TODO return (false, nil) to continue with next phase
		return true, nil
	}
	return false, nil
}

func (c *Controller) handlePipelineRunWaiting(
	ctx context.Context,
	runManager run.Manager,
	pipelineRun k8s.PipelineRun,
	pipelineRunsConfig *cfg.PipelineRunsConfigStruct,
) (bool, error) {
	ctx, logger := extendContextLoggerWithPipelineRunInfo(ctx, pipelineRun.GetAPIObject())

	if pipelineRun.GetStatus().State == api.StateWaiting {
		logger.V(3).Info("Waiting for pipeline execution")

		run, err := runManager.GetRun(ctx, pipelineRun)
		if err != nil {
			return true, c.onGetRunError(ctx, pipelineRun, err, api.StateCleaning, api.ResultErrorInfra, errorMessageWaitingFailed)
		}

		// Check for wait timeout
		startTime := pipelineRun.GetStatus().StateDetails.StartedAt
		timeout := c.getWaitTimeout(pipelineRunsConfig)
		if startTime.Add(timeout.Duration).Before(time.Now()) {
			err := fmt.Errorf(
				"main pod has not started after %s",
				timeout.Duration,
			)
			return true, c.handleResultError(ctx, pipelineRun, api.ResultErrorInfra, errorMessageWaitingFailed, err)
		}

		if run == nil {
			return true, c.startPipelineRun(ctx, runManager, pipelineRun, pipelineRunsConfig)
		} else if run.IsRestartable() {
			c.eventRecorder.Event(pipelineRun.GetReference(), corev1.EventTypeWarning, api.EventReasonWaitingFailed, "restarting")
			return c.restart(ctx, runManager, pipelineRun)
		}

		started := run.GetStartTime()
		if started != nil {
			if err := c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StateRunning, *started); err != nil {
				return true, err
			}
		}

		// TODO return (false, nil) to continue with next phase
		return true, nil
	}
	return false, nil
}

func (c *Controller) startPipelineRun(ctx context.Context,
	runManager run.Manager,
	pipelineRun k8s.PipelineRun,
	pipelineRunsConfig *cfg.PipelineRunsConfigStruct) error {
	if err := runManager.Start(ctx, pipelineRun, pipelineRunsConfig); err != nil {
		c.eventRecorder.Event(pipelineRun.GetReference(), corev1.EventTypeWarning, api.EventReasonWaitingFailed, err.Error())
		resultClass := serrors.GetClass(err)
		// In case we have a result we can cleanup. Otherwise we retry in the next iteration.
		if resultClass != api.ResultUndefined {
			return c.handleResultError(ctx, pipelineRun, resultClass, errorMessageWaitingFailed, err)
		}
		return err
	}
	return nil
}

func (c *Controller) restart(
	ctx context.Context,
	runManager run.Manager,
	pipelineRun k8s.PipelineRun,
) (bool, error) {
	if err := runManager.DeleteRun(ctx, pipelineRun); err != nil {
		if serrors.IsRecoverable(err) {
			return true, err
		}
		return true, c.handleResultError(ctx, pipelineRun, api.ResultErrorInfra, "run deletion for restart failed", err)
	}
	return true, nil
}

func (c *Controller) handlePipelineRunRunning(
	ctx context.Context,
	runManager run.Manager,
	pipelineRun k8s.PipelineRun,
	pipelineRunsConfig *cfg.PipelineRunsConfigStruct,
) (bool, error) {
	ctx, logger := extendContextLoggerWithPipelineRunInfo(ctx, pipelineRun.GetAPIObject())

	if pipelineRun.GetStatus().State == api.StateRunning {
		logger.V(3).Info("Examining running pipeline")

		run, err := runManager.GetRun(ctx, pipelineRun)
		if err != nil {
			return true, c.onGetRunError(ctx, pipelineRun, err, api.StateCleaning, api.ResultErrorInfra, errorMessageRunningFailed)
		}
		if run == nil {
			err = fmt.Errorf("task run not found in namespace %q", pipelineRun.GetRunNamespace())
			return true, c.onGetRunError(ctx, pipelineRun, err, api.StateCleaning, api.ResultErrorInfra, errorMessageRunningFailed)
		}

		containerInfo := run.GetContainerInfo()
		pipelineRun.UpdateContainer(ctx, containerInfo)
		if finished, result := run.IsFinished(); finished {
			pipelineRun.UpdateMessage(run.GetMessage())
			return true, c.updateStateAndResult(ctx, pipelineRun, api.StateCleaning, result, *run.GetCompletionTime())
		}
		// commit container update
		err = c.commitStatusAndMeter(ctx, pipelineRun)
		if err != nil {
			return true, err
		}

		// TODO return (false, nil) to continue with next phase
		return true, nil
	}
	return false, nil
}

func (c *Controller) handlePipelineRunCleaning(
	ctx context.Context,
	runManager run.Manager,
	pipelineRun k8s.PipelineRun,
) (bool, error) {
	ctx, logger := extendContextLoggerWithPipelineRunInfo(ctx, pipelineRun.GetAPIObject())

	if pipelineRun.GetStatus().State == api.StateCleaning {
		logger.V(3).Info("Cleaning up pipeline execution")

		err := runManager.Cleanup(ctx, pipelineRun)
		if err != nil {
			c.eventRecorder.Event(pipelineRun.GetReference(), corev1.EventTypeWarning, api.EventReasonCleaningFailed, err.Error())
			return true, err
		}
		if err = c.changeAndCommitStateAndMeter(ctx, pipelineRun, api.StateFinished, metav1.Now()); err != nil {
			return true, err
		}
		if err = pipelineRun.DeleteFinalizerAndCommitIfExists(ctx); err != nil {
			return true, err
		}
	}
	return false, nil
}

func (c *Controller) getWaitTimeout(pipelineRunsConfig *cfg.PipelineRunsConfigStruct) *metav1.Duration {
	timeout := pipelineRunsConfig.TimeoutWait
	if utils.IsZeroDuration(timeout) {
		timeout = utils.Metav1Duration(time.Duration(defaultWaitTimeout))
	}
	return timeout
}

// TODO find better name
func (c *Controller) handleResultError(ctx context.Context, pipelineRun k8s.PipelineRun, result api.Result, message string, err error) error {
	logger := klog.FromContext(ctx)
	logger.Info("Updating error message to pipeline run",
		"message", utils.Trim(message),
		"errorMessage", err,
	)
	pipelineRun.StoreErrorAsMessage(ctx, err, message)
	return c.updateStateAndResult(ctx, pipelineRun, api.StateCleaning, result, metav1.Now())
}

// TODO change name to express semantics
// This method is not only called when pipelineManager.GetRun()
// failed, but also in other context.
func (c *Controller) onGetRunError(
	ctx context.Context,
	pipelineRun k8s.PipelineRun,
	err error,
	targetState api.State,
	result api.Result,
	message string,
) error {
	logger := klog.FromContext(ctx)
	logger.Error(err, message)

	c.eventRecorder.Event(pipelineRun.GetReference(), corev1.EventTypeWarning, api.EventReasonRunningFailed, err.Error())
	if serrors.IsRecoverable(err) {
		return err
	}
	pipelineRun.StoreErrorAsMessage(ctx, err, message)
	return c.updateStateAndResult(ctx, pipelineRun, targetState, result, metav1.Now())
}

func (c *Controller) changeAndCommitStateAndMeter(ctx context.Context, pipelineRun k8s.PipelineRun, state api.State, ts metav1.Time) error {
	if err := c.changeState(ctx, pipelineRun, state, ts); err != nil {
		return err
	}
	return c.commitStatusAndMeter(ctx, pipelineRun)
}

func (c *Controller) updateStateAndResult(ctx context.Context, pipelineRun k8s.PipelineRun, state api.State, result api.Result, ts metav1.Time) error {
	pipelineRun.UpdateResult(ctx, result, ts)
	if err := c.changeAndCommitStateAndMeter(ctx, pipelineRun, state, ts); err != nil {
		return err
	}
	metrics.PipelineRunsResult.Observe(pipelineRun.GetStatus().Result)
	if state == api.StateFinished {
		return pipelineRun.DeleteFinalizerAndCommitIfExists(ctx)
	}
	return nil
}

func (c *Controller) commitStatusAndMeter(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	logger := klog.FromContext(ctx)

	start := time.Now()
	finishedStates, err := pipelineRun.CommitStatus(ctx)
	if err != nil {
		logger.V(6).Info("Failed to commit pipeline run status", "err", err.Error())
		return err
	}
	end := time.Now()
	elapsed := end.Sub(start)
	logger.V(6).Info("Completed committing pipeline run status", "duration", elapsed)
	metrics.UpdatesLatency.Observe("UpdateState", elapsed)
	for _, finishedState := range finishedStates {
		metrics.PipelineRunsStateFinished.Observe(finishedState)
	}
	return nil
}

// handlePipelineRunAbort checks if pipeline run should be aborted.
// If the user requested abortion it updates message, result and state
// to trigger a cleanup.
func (c *Controller) handlePipelineRunAbort(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	ctx, _ = extendContextLoggerWithPipelineRunInfo(ctx, pipelineRun.GetAPIObject())

	intent := pipelineRun.GetSpec().Intent
	if intent == api.IntentAbort && pipelineRun.GetStatus().Result == api.ResultUndefined {
		pipelineRun.UpdateMessage("Aborted")
		return c.updateStateAndResult(ctx, pipelineRun, api.StateCleaning, api.ResultAborted, metav1.Now())
	}
	return nil
}

func (c *Controller) addToWorkqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
	c.logger.V(4).Info("Added item to workqueue", "key", key)
}

// addToWorkqueueFromAssociated takes any resource implementing metav1.Object and attempts
// to find the PipelineRun resource that 'owns' it. It does this by looking for
// a specific annotation. If such annotation exists, the named PipelineRun
// is put into the controller's work queue to be processed.
func (c *Controller) addToWorkqueueFromAssociated(obj interface{}) {
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
		c.logger.V(3).Info("Recovered deleted object from unknown state",
			"object", klog.KObj(object))
	}
	c.logger.V(4).Info("Deriving workqueue item from associated object", "object", klog.KObj(object))

	runKey := runmgr.GetPipelineRunKeyAnnotation(object)
	if runKey != "" {
		c.workqueue.Add(runKey)
		c.logger.V(4).Info("Added item to workqueue", "key", runKey)
	}
}
