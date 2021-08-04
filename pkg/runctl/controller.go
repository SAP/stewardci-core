/*
based on sample-controller from https://github.com/kubernetes/sample-controller/blob/7047ee6ceceef2118a2017bbfff4a86c1f56f1ca/controller.go
*/

package runctl

import (
	"fmt"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/client/clientset/versioned/scheme"
	"github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"github.com/SAP/stewardci-core/pkg/maintenancemode"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	run "github.com/SAP/stewardci-core/pkg/runctl/run"
	utils "github.com/SAP/stewardci-core/pkg/utils"
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

const kind = "PipelineRuns"

// Used for logging (control loop) "still alive" messages
var heartbeatIntervalSeconds int64 = 60
var heartbeatTimer int64 = 0

// Interval for histogram creation set to prometheus default scrape interval
var meteringInterval = time.Minute * 1

// Controller processes PipelineRun resources
type Controller struct {
	factory              k8s.ClientFactory
	pipelineRunFetcher   k8s.PipelineRunFetcher
	pipelineRunSynced    cache.InformerSynced
	tektonTaskRunsSynced cache.InformerSynced
	workqueue            workqueue.RateLimitingInterface
	metrics              metrics.Metrics
	testing              *controllerTesting
	recorder             record.EventRecorder
	pipelineRunLister    v1alpha1.PipelineRunLister
	pipelineRunStore     cache.Store
}

type controllerTesting struct {
	runManagerStub             run.Manager
	newRunManagerStub          func(k8s.ClientFactory, secrets.SecretProvider) run.Manager
	loadPipelineRunsConfigStub func() (*cfg.PipelineRunsConfigStruct, error)
	isMaintenanceModeStub      func() (bool, error)
}

// NewController creates new Controller
func NewController(factory k8s.ClientFactory, metrics metrics.Metrics) *Controller {
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
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), kind),
		metrics:              metrics,
		recorder:             recorder,
		pipelineRunStore:     pipelineRunInformer.Informer().GetStore(),
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

// meterPipelineRuns observes certain metrics of all existing pipeline runs (in the informer cache).
func (c *Controller) meterPipelineRuns() {
	klog.V(4).Infof("metering all pipeline runs")
	objs := c.pipelineRunStore.List()
	for _, obj := range objs {
		pipelineRun := obj.(*api.PipelineRun)

		// do not meter delays caused by finalizers
		if pipelineRun.DeletionTimestamp.IsZero() {
			err := c.metrics.ObserveOngoingStateDuration(pipelineRun)
			if err != nil {
				klog.Errorf("metrics observation of PipelineRun %v failed: %v", pipelineRun, err)
			}
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
	go wait.Until(c.meterPipelineRuns, meteringInterval, stopCh)

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
		now := time.Now().Unix()
		if heartbeatTimer <= now-heartbeatIntervalSeconds {
			heartbeatTimer = now
			klog.V(3).Infof("Run Controller still alive")
		}
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
		klog.V(4).Infof("process %s queue length: %d", key, c.workqueue.Len())
		c.metrics.SetQueueCount(c.workqueue.Len())

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

func (c *Controller) changeState(pipelineRun k8s.PipelineRun, state api.State) error {
	err := pipelineRun.UpdateState(state)
	if err != nil {
		klog.V(3).Infof("Failed to UpdateState of [%s] to %q: %q", pipelineRun.String(), state, err.Error())
		return err
	}

	return nil
}

func (c *Controller) createRunManager(pipelineRun k8s.PipelineRun) run.Manager {
	if c.testing != nil && c.testing.runManagerStub != nil {
		return c.testing.runManagerStub
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

func (c *Controller) loadPipelineRunsConfig() (*cfg.PipelineRunsConfigStruct, error) {
	if c.testing != nil && c.testing.loadPipelineRunsConfigStub != nil {
		return c.testing.loadPipelineRunsConfigStub()
	}
	return cfg.LoadPipelineRunsConfig(c.factory)
}

func (c *Controller) isMaintenanceMode() (bool, error) {
	if c.testing != nil && c.testing.isMaintenanceModeStub != nil {
		return c.testing.isMaintenanceModeStub()
	}
	return maintenancemode.IsMaintenanceMode(c.factory)
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Initial checks on cached pipelineRun
	pipelineRunAPIObj, err := c.pipelineRunFetcher.ByKey(key)
	if err != nil {
		return err
	}
	// If pipelineRun is not found there is nothing to sync
	if pipelineRunAPIObj == nil {
		return nil
	}
	// fast exit
	if pipelineRunAPIObj.GetDeletionTimestamp().IsZero() {
		if pipelineRunAPIObj.Status.State == api.StateFinished {
			return nil
		}
	} else {
		if !utils.StringSliceContains(pipelineRunAPIObj.ObjectMeta.Finalizers, k8s.FinalizerName) {
			return nil
		}
	}

	// Get real pipelineRun bypassing cache
	pipelineRun, err := k8s.NewPipelineRun(pipelineRunAPIObj, c.factory)
	if err != nil {
		return err
	}

	// If pipelineRun is not found there is nothing to sync
	if pipelineRun == nil {
		return nil
	}

	// Check if object has deletion timestamp
	// If not, try to add finalizer if missing
	if pipelineRun.HasDeletionTimestamp() {
		runManager := c.createRunManager(pipelineRun)
		err = runManager.Cleanup(pipelineRun)
		if err == nil {
			err = pipelineRun.UpdateResult(api.ResultDeleted)
			if err != nil {
				return err
			}
			err = c.finish(pipelineRun)
			if err == nil {
				c.metrics.CountResult(api.ResultDeleted)
			}
		}
		return err
	}
	pipelineRun.AddFinalizer()

	// Finished and no deletion timestamp, no need to process anything further
	if pipelineRun.GetStatus().State == api.StateFinished {
		return nil
	}

	// Check if pipeline run is aborted
	if err := c.handleAborted(pipelineRun); err != nil {
		return err
	}

	// As soon as we have a result we can cleanup
	if pipelineRun.GetStatus().Result != api.ResultUndefined && pipelineRun.GetStatus().State != api.StateCleaning {
		err = c.changeState(pipelineRun, api.StateCleaning)
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
		maintenanceMode, err := c.isMaintenanceMode()
		if err != nil {
			return err
		}
		if maintenanceMode {
			err := fmt.Errorf("pipeline execution is paused while the system is in maintenance mode")
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeNormal, api.EventReasonMaintenanceMode, err.Error())
			// Return error that the pipeline stays in the queue and will be processed after switching back to normal mode.
			return err
		}
		if err = c.changeState(pipelineRun, api.StatePreparing); err != nil {
			return err
		}
		if err = c.commitAndMeter(pipelineRun); err != nil {
			return err
		}
		c.metrics.CountStart()
	}

	// the configuration should be loaded once per sync to avoid inconsistencies
	// in case of concurrent configuration changes
	pipelineRunsConfig, err := c.loadPipelineRunsConfig()
	if err != nil {
		if serrors.IsRecoverable(err) {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonLoadPipelineRunsConfigFailed, err.Error())
			return err
		}
		pipelineRun.UpdateResult(api.ResultErrorInfra)
		pipelineRun.StoreErrorAsMessage(err, "failed to load configuration for pipeline runs")
		c.metrics.CountResult(pipelineRun.GetStatus().Result)

		return c.finish(pipelineRun)
	}

	runManager := c.createRunManager(pipelineRun)

	// Process pipeline run based on current state
	switch state := pipelineRun.GetStatus().State; state {
	case api.StatePreparing:
		err = runManager.Start(pipelineRun, pipelineRunsConfig)
		if err != nil {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonPreparingFailed, err.Error())
			resultClass := serrors.GetClass(err)
			// In case we have a result we can cleanup. Otherwise we retry in the next iteration.
			if resultClass != api.ResultUndefined {
				pipelineRun.UpdateMessage(err.Error())
				pipelineRun.UpdateResult(resultClass)
				if errClean := c.changeState(pipelineRun, api.StateCleaning); errClean != nil {
					return errClean
				}
				pipelineRun.StoreErrorAsMessage(err, "preparing failed")
				if err := c.commitAndMeter(pipelineRun); err != nil {
					return err
				}
				c.metrics.CountResult(pipelineRun.GetStatus().Result)
				return nil
			}
			return err
		}
		if err = c.changeState(pipelineRun, api.StateWaiting); err != nil {
			return err
		}
		// TODO try to get rid of that commit
		if err := c.commitAndMeter(pipelineRun); err != nil {
			return err
		}
	case api.StateWaiting:
		run, err := runManager.GetRun(pipelineRun)
		if err != nil {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonWaitingFailed, err.Error())
			if serrors.IsRecoverable(err) {
				return err
			}
			if errClean := c.changeState(pipelineRun, api.StateCleaning); errClean != nil {
				return errClean
			}
			pipelineRun.StoreErrorAsMessage(err, "waiting failed")
			pipelineRun.UpdateResult(api.ResultErrorInfra)
			if err := c.commitAndMeter(pipelineRun); err != nil {
				return err
			}
			c.metrics.CountResult(api.ResultErrorInfra)
			return nil
		}
		started := run.GetStartTime()
		if started != nil {
			if err = c.changeState(pipelineRun, api.StateRunning); err != nil {
				return err
			}
			if err = c.commitAndMeter(pipelineRun); err != nil {
				return err
			}
		}
	case api.StateRunning:
		run, err := runManager.GetRun(pipelineRun)
		if err != nil {
			c.recorder.Event(pipelineRunAPIObj, corev1.EventTypeWarning, api.EventReasonRunningFailed, err.Error())
			if serrors.IsRecoverable(err) {
				return err
			}
			if errClean := c.changeState(pipelineRun, api.StateCleaning); errClean != nil {
				return errClean
			}
			pipelineRun.StoreErrorAsMessage(err, "running failed")
			return c.commitAndMeter(pipelineRun)
		}
		containerInfo := run.GetContainerInfo()
		pipelineRun.UpdateContainer(containerInfo)
		if finished, result := run.IsFinished(); finished {
			msg := run.GetMessage()
			pipelineRun.UpdateMessage(msg)
			pipelineRun.UpdateResult(result)
			if err = c.changeState(pipelineRun, api.StateCleaning); err != nil {
				return err
			}
			if err = c.commitAndMeter(pipelineRun); err != nil {
				return err
			}
			c.metrics.CountResult(result)
		}
	case api.StateCleaning:
		err = runManager.Cleanup(pipelineRun)
		return c.finish(pipelineRun)
	default:
		klog.V(2).Infof("Skip PipelineRun with state %s", pipelineRun.GetStatus().State)
	}
	return nil
}

func (c *Controller) commitAndMeter(pipelineRun k8s.PipelineRun) error {
	finishedStates, err := pipelineRun.CommitStatus()
	if err != nil {
		return err
	}
	for _, finishedState := range finishedStates {
		err := c.metrics.ObserveDurationByState(finishedState)
		if err != nil {
			klog.Errorf("Failed to measure state '%+v': '%s'", finishedState, err)
		}
	}
	return nil
}

func (c *Controller) finish(pipelineRun k8s.PipelineRun) error {
	if err := c.changeState(pipelineRun, api.StateFinished); err != nil {
		return err
	}
	if err := c.commitAndMeter(pipelineRun); err != nil {
		return err
	}
	return pipelineRun.DeleteFinalizerIfExists()
}

// handleAborted checks if pipeline run should be aborted.
// If the user requested abortion it updates message, result and state
// to trigger a cleanup.
func (c *Controller) handleAborted(pipelineRun k8s.PipelineRun) error {
	intent := pipelineRun.GetSpec().Intent
	if intent == api.IntentAbort && pipelineRun.GetStatus().Result == api.ResultUndefined {
		pipelineRun.UpdateMessage("Aborted")
		pipelineRun.UpdateResult(api.ResultAborted)
		return c.changeState(pipelineRun, api.StateCleaning)
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
