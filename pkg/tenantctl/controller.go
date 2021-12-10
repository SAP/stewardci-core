/*
based on sample-controller from https://github.com/kubernetes/sample-controller/blob/7047ee6ceceef2118a2017bbfff4a86c1f56f1ca/controller.go
*/

package tenantctl

import (
	"context"
	"fmt"
	"time"

	stewardapis "github.com/SAP/stewardci-core/pkg/apis/steward"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardv1alpha1listers "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	slabels "github.com/SAP/stewardci-core/pkg/stewardlabels"
	metrics "github.com/SAP/stewardci-core/pkg/tenantctl/metrics"
	utils "github.com/SAP/stewardci-core/pkg/utils"
	errors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	equality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
	cache "k8s.io/client-go/tools/cache"
	workqueue "k8s.io/client-go/util/workqueue"
	klog "k8s.io/klog/v2"
	knativeapis "knative.dev/pkg/apis"
)

const (
	tenantNamespaceRoleBindingNamePrefix = stewardapis.GroupName + "--tenant-role-binding-"

	// heartbeatStimulusKey is a special key inserted into the controller
	// work queue as heartbeat stimulus.
	// It is an invalid Kubernetes name to avoid conflicts with real
	// pipeline runs.
	heartbeatStimulusKey = "Heartbeat Stimulus"
)

// Controller for Steward Tenants
type Controller struct {
	factory      k8s.ClientFactory
	fetcher      k8s.TenantFetcher
	tenantSynced cache.InformerSynced
	tenantLister stewardv1alpha1listers.TenantLister
	workqueue    workqueue.RateLimitingInterface
	syncCount    int64
	testing      *controllerTesting

	heartbeatInterval time.Duration
	heartbeatLogLevel *klog.Level
}

type controllerTesting struct {
	createRoleBindingStub          func(roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)
	getClientConfigStub            func(factory k8s.ClientFactory, clientNamespace string) (clientConfig, error)
	listManagedRoleBindingsStub    func(namespace string) (*rbacv1.RoleBindingList, error)
	reconcileTenantRoleBindingStub func(tenant *stewardv1alpha1.Tenant, namespace string, config clientConfig) (bool, error)
	updateStatusStub               func(tenant *stewardv1alpha1.Tenant) (*stewardv1alpha1.Tenant, error)
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
	informer := factory.StewardInformerFactory().Steward().V1alpha1().Tenants()
	fetcher := k8s.NewListerBasedTenantFetcher(informer.Lister())

	controller := &Controller{
		factory:      factory,
		fetcher:      fetcher,
		tenantSynced: informer.Informer().HasSynced,
		tenantLister: informer.Lister(),
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), metrics.WorkqueueName),
	}

	controller.heartbeatInterval = opts.HeartbeatInterval
	if opts.HeartbeatLogLevel != nil {
		copyOfValue := *opts.HeartbeatLogLevel
		controller.heartbeatLogLevel = &copyOfValue
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.onTenantAdd,
		UpdateFunc: controller.onTenantUpdate,
		DeleteFunc: controller.onTenantDelete,
	})
	return controller
}

func (c *Controller) getSyncCount() int64 {
	return c.syncCount
}

func (c *Controller) getNamespaceManager(config clientConfig) k8s.NamespaceManager {
	return k8s.NewNamespaceManager(
		c.factory,
		config.GetTenantNamespacePrefix(),
		config.GetTenantNamespaceSuffixLength(),
	)
}

// Run runs the controller.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(2).Infof("Sync cache")
	if ok := cache.WaitForCacheSync(stopCh, c.tenantSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

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
	klog.V(2).Infof("Workers running [%v]", threadiness)

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

	numRequeues := c.workqueue.NumRequeues(obj)
	if numRequeues > 0 {
		klog.V(4).Infof("Requeued %v times '%s'", numRequeues, obj.(string))
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
			// (The delay in case of multiple retries will increase exponentially)
			c.workqueue.AddRateLimited(obj)
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

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the tenant resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {

	if key == heartbeatStimulusKey {
		c.heartbeat()
		return nil
	}

	ctx := context.Background()

	origTenant, err := c.fetcher.ByKey(ctx, key)
	if err != nil {
		return err
	}

	if origTenant == nil {
		return nil
	}

	tenant := origTenant.DeepCopy()

	klog.V(4).Infof(c.formatLog(tenant, "started reconciliation"))
	if klog.V(4).Enabled() {
		defer klog.V(4).Infof(c.formatLog(&stewardv1alpha1.Tenant{ObjectMeta: *tenant.ObjectMeta.DeepCopy()}, "finished reconciliation"))
	}

	// the configuration should be loaded once per sync to avoid inconsistencies
	// in case of concurrent configuration changes
	config, err := c.getClientConfig(ctx, c.factory, tenant.GetNamespace())
	if err != nil {
		klog.Infof(c.formatLog(tenant), err)
		return err
	}

	if !tenant.ObjectMeta.DeletionTimestamp.IsZero() {
		klog.V(3).Infof(c.formatLog(tenant, "tenant is marked as deleted"))
		if !c.hasFinalizer(tenant) {
			klog.V(3).Infof(c.formatLog(tenant, "dependent resources cleaned already, nothing to do"))
			return nil
		}
		err = c.deleteTenantNamespace(ctx, tenant.Status.TenantNamespaceName, tenant, config)
		if err != nil {
			return err
		}
		_, err = c.removeFinalizerAndUpdate(ctx, tenant)
		if err == nil {
			c.syncCount++
		}
		return err
	}

	tenant, err = c.addFinalizerAndUpdate(ctx, tenant)
	if err != nil {
		return err
	}

	reconcileErr := c.reconcile(ctx, config, tenant)

	// do not update the status if there's no change
	if !equality.Semantic.DeepEqual(origTenant.Status, tenant.Status) {
		if _, err := c.updateStatus(ctx, tenant); err != nil {
			if !c.isInitialized(origTenant) && c.isInitialized(tenant) {
				c.deleteTenantNamespace(ctx, tenant.Status.TenantNamespaceName, tenant, config) // clean-up ignoring error
			}
			return err
		}
	}

	if reconcileErr != nil {
		return reconcileErr
	}

	c.updateMetrics()
	c.syncCount++
	return nil
}

func (c *Controller) isInitialized(tenant *stewardv1alpha1.Tenant) bool {
	return tenant.Status.TenantNamespaceName != ""
}

func (c *Controller) reconcile(ctx context.Context, config clientConfig, tenant *stewardv1alpha1.Tenant) (err error) {
	if c.isInitialized(tenant) {
		err = c.reconcileInitialized(ctx, config, tenant)
	} else {
		err = c.reconcileUninitialized(ctx, config, tenant)
	}
	return
}

func (c *Controller) reconcileUninitialized(ctx context.Context, config clientConfig, tenant *stewardv1alpha1.Tenant) error {
	klog.V(3).Infof(c.formatLog(tenant, "tenant not initialized yet"))

	nsName, err := c.createTenantNamespace(ctx, config, tenant)
	if err != nil {
		condMsg := "Failed to create a new tenant namespace."
		tenant.Status.SetCondition(&knativeapis.Condition{
			Type:    knativeapis.ConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  stewardv1alpha1.StatusReasonFailed,
			Message: condMsg,
		})
		return err
	}

	_, err = c.reconcileTenantRoleBinding(ctx, tenant, nsName, config)
	if err != nil {
		condMsg := "Failed to initialize a new tenant namespace because the RoleBinding could not be created."
		tenant.Status.SetCondition(&knativeapis.Condition{
			Type:    knativeapis.ConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  stewardv1alpha1.StatusReasonFailed,
			Message: condMsg,
		})
		c.deleteTenantNamespace(ctx, nsName, tenant, config) // clean-up ignoring error
		return err
	}

	tenant.Status.TenantNamespaceName = nsName

	tenant.Status.SetCondition(&knativeapis.Condition{
		Type:   knativeapis.ConditionReady,
		Status: corev1.ConditionTrue,
	})

	return nil
}

func (c *Controller) reconcileInitialized(ctx context.Context, config clientConfig, tenant *stewardv1alpha1.Tenant) error {
	klog.V(4).Infof(c.formatLog(tenant, "tenant is initialized already"))

	nsName := tenant.Status.TenantNamespaceName

	exists, err := c.checkNamespaceExists(ctx, nsName)
	if err != nil {
		klog.Infof(c.formatLog(tenant), err)
		return err
	}

	if !exists {
		condMsg := fmt.Sprintf(
			"The tenant namespace %q does not exist anymore."+
				" This issue must be analyzed and fixed by an operator.",
			nsName,
		)
		tenant.Status.SetCondition(&knativeapis.Condition{
			Type:    knativeapis.ConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  stewardv1alpha1.StatusReasonDependentResourceState,
			Message: condMsg,
		})
		err = errors.Errorf("tenant namespace %q does not exist anymore", nsName)
		klog.V(3).Infof(c.formatLog(tenant), err)
		return err
	}

	needForUpdateDetected, err := c.reconcileTenantRoleBinding(ctx, tenant, nsName, config)
	if err != nil {
		if needForUpdateDetected {
			condMsg := fmt.Sprintf(
				"The RoleBinding in tenant namespace %q is outdated but could not be updated.",
				nsName,
			)
			tenant.Status.SetCondition(&knativeapis.Condition{
				Type:    knativeapis.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  stewardv1alpha1.StatusReasonDependentResourceState,
				Message: condMsg,
			})
		}
		return err
	}

	tenant.Status.SetCondition(&knativeapis.Condition{
		Type:   knativeapis.ConditionReady,
		Status: corev1.ConditionTrue,
	})

	return nil
}

func (c *Controller) getClientConfig(ctx context.Context, factory k8s.ClientFactory, clientNamespace string) (clientConfig, error) {
	if c.testing != nil && c.testing.getClientConfigStub != nil {
		return c.testing.getClientConfigStub(factory, clientNamespace)
	}
	return getClientConfig(ctx, factory, clientNamespace)
}

func (c *Controller) hasFinalizer(tenant *stewardv1alpha1.Tenant) bool {
	return utils.StringSliceContains(tenant.GetFinalizers(), k8s.FinalizerName)
}

func (c *Controller) addFinalizerAndUpdate(ctx context.Context, tenant *stewardv1alpha1.Tenant) (*stewardv1alpha1.Tenant, error) {
	changed, finalizerList := utils.AddStringIfMissing(tenant.GetFinalizers(), k8s.FinalizerName)
	if changed {
		tenant.SetFinalizers(finalizerList)
		return c.update(ctx, tenant)
	}
	return tenant, nil
}

func (c *Controller) removeFinalizerAndUpdate(ctx context.Context, tenant *stewardv1alpha1.Tenant) (*stewardv1alpha1.Tenant, error) {
	changed, finalizerList := utils.RemoveString(tenant.GetFinalizers(), k8s.FinalizerName)
	if changed {
		tenant.SetFinalizers(finalizerList)
		return c.update(ctx, tenant)
	}
	return tenant, nil
}

func (c *Controller) updateStatus(ctx context.Context, tenant *stewardv1alpha1.Tenant) (*stewardv1alpha1.Tenant, error) {
	if c.testing != nil && c.testing.updateStatusStub != nil {
		return c.testing.updateStatusStub(tenant)
	}

	client := c.factory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	updatedTenant, err := client.UpdateStatus(ctx, tenant, metav1.UpdateOptions{})
	if err != nil {
		err = errors.WithMessage(err, "failed to update resource status")
		klog.V(3).Infof(c.formatLog(tenant), err)
		return nil, err
	}
	return updatedTenant, nil
}

func (c *Controller) update(ctx context.Context, tenant *stewardv1alpha1.Tenant) (*stewardv1alpha1.Tenant, error) {
	client := c.factory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	result, err := client.Update(ctx, tenant, metav1.UpdateOptions{})
	if err != nil {
		err = errors.WithMessagef(err,
			"failed to update tenant %q in namespace %q",
			tenant.GetName(), tenant.GetNamespace(),
		)
		klog.V(3).Infof(c.formatLog(tenant), err)
		return tenant, err
	}
	return result, nil
}

func (c *Controller) checkNamespaceExists(ctx context.Context, name string) (bool, error) {
	namespaces := c.factory.CoreV1().Namespaces()
	namespace, err := namespaces.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		err = errors.WithMessagef(err, "error: failed to get namespace %q", name)
		return false, err
	}
	return namespace.GetDeletionTimestamp().IsZero(), nil
}

func (c *Controller) createTenantNamespace(ctx context.Context, config clientConfig, tenant *stewardv1alpha1.Tenant) (string, error) {
	klog.V(4).Infof(c.formatLog(tenant, "creating new tenant namespace"))
	namespaceManager := c.getNamespaceManager(config)
	nsName, err := namespaceManager.Create(ctx, tenant.GetName(), nil)
	if err != nil {
		err = errors.WithMessage(err, "failed to create new tenant namespace")
		klog.V(4).Infof(c.formatLog(tenant), err)
		return "", err
	}
	return nsName, err
}

func (c *Controller) deleteTenantNamespace(ctx context.Context, namespace string, tenant *stewardv1alpha1.Tenant, config clientConfig) error {
	if namespace == "" {
		return nil
	}
	klog.V(4).Infof(c.formatLogf(tenant, "rolling back tenant namespace %q", namespace))
	namespaceManager := c.getNamespaceManager(config)
	err := namespaceManager.Delete(ctx, namespace)
	if err != nil {
		err = errors.WithMessagef(err, "failed to delete tenant namespace %q", namespace)
		klog.V(4).Infof(c.formatLog(tenant), err)
		return err
	}
	return nil
}

/*
reconcileTenantRoleBinding compares the actual state of the role binding
in the tenant namespace with the desired state.
In case of a mismatch it tries to achieve the desired state by replacing
the existing role binding resource object(s) by a new one.

Output parameter `needForUpdateDetected` indicates whether the need for an
update has been detected, and will be set accordingly both in case of success
and error. In case of success (err == nil) it indicates whether an update
_has been_ performed. In case of error (err != nil) a value of `true`
indicates that an update _would have_ to be performed (and maybe has been
done partially), while a value of `false` indicates that _it is unknown_
whether an update is necessary (due to the error).
*/
func (c *Controller) reconcileTenantRoleBinding(ctx context.Context, tenant *stewardv1alpha1.Tenant, namespace string, config clientConfig) (needForUpdateDetected bool, err error) {
	if c.testing != nil && c.testing.reconcileTenantRoleBindingStub != nil {
		return c.testing.reconcileTenantRoleBindingStub(tenant, namespace, config)
	}

	/*
		The roleRef of an existing RoleBinding cannot be updated (prohibited by
		server). This means we need to create a new RoleBinding when we want to
		change the tenant role.
		We cannot simply delete the existing role and create a new one (in that
		order), because that would revoke permissions for some time and may fail
		API calls concurrently performed by the respective subjects.
		Instead we will first create the new RoleBinding and remove the old one
		afterwards.
		To deal with remainders from previously failed attempts, we expect that
		an arbitrary number of RoleBinding objects may exist. Recreation takes
		place if the number of existing role bindings is not exactly one, or the
		single existing role binding is not up-to-date.
		We manage only those RoleBinding objects that are marked as "managed by
		Steward". All others will not be touched or taken into account.
	*/

	err = func() error {
		rbList, err := c.listManagedRoleBindings(ctx, namespace)
		if err != nil {
			return err
		}

		clientNamespace := tenant.GetNamespace()
		expectedTenantRB := c.generateTenantRoleBinding(namespace, clientNamespace, config)

		if len(rbList.Items) != 1 || !c.isTenantRoleBindingUpToDate(&rbList.Items[0], expectedTenantRB) {
			needForUpdateDetected = true
		}

		if needForUpdateDetected {
			klog.V(4).Infof(c.formatLogf(tenant, "updating RoleBinding in tenant namespace %q", namespace))
			_, err = c.createRoleBinding(ctx, expectedTenantRB)
			if err != nil {
				return err
			}
			err = c.deleteRoleBindingsFromList(ctx, rbList)
			if err != nil {
				return err
			}
		}

		return nil
	}()

	if err != nil {
		err = errors.WithMessagef(err,
			"failed to reconcile the RoleBinding in tenant namespace %q",
			namespace,
		)
		klog.V(4).Infof(c.formatLog(tenant), err)
	}
	return
}

/**
 * generateTenantRoleBinding generates the role binding for a tenant namespace
 * as in-memory object only (no persistence in K8s).
 */
func (c *Controller) generateTenantRoleBinding(
	tenantNamespace string, clientNamespace string, config clientConfig,
) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			// let the server generate a unique name
			GenerateName: tenantNamespaceRoleBindingNamePrefix,
			Namespace:    tenantNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     string(config.GetTenantRoleName()),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: tenantNamespace,
				Name:      "default",
			},
			// The client service account should have access to all tenant namespaces
			// of this client.
			// For stricter tenant isolation we might offer an opt-out in the future
			// so that access to tenant namespaces requires using the respective
			// tenant service account.
			{
				Kind:      "ServiceAccount",
				Namespace: clientNamespace,
				Name:      "default",
			},
		},
	}

	slabels.LabelAsSystemManaged(roleBinding)

	return roleBinding
}

func (c *Controller) isTenantRoleBindingUpToDate(current *rbacv1.RoleBinding, expected *rbacv1.RoleBinding) bool {
	return true &&
		equality.Semantic.DeepEqual(expected.GetLabels(), current.GetLabels()) &&
		equality.Semantic.DeepEqual(expected.GetAnnotations(), current.GetAnnotations()) &&
		equality.Semantic.DeepEqual(expected.RoleRef, current.RoleRef) &&
		equality.Semantic.DeepEqual(expected.Subjects, current.Subjects)
}

func (c *Controller) listManagedRoleBindings(ctx context.Context, namespace string) (*rbacv1.RoleBindingList, error) {
	if c.testing != nil && c.testing.listManagedRoleBindingsStub != nil {
		return c.testing.listManagedRoleBindingsStub(namespace)
	}

	roleBindingIfc := c.factory.RbacV1().RoleBindings(namespace)
	listOptions := metav1.ListOptions{
		LabelSelector: stewardv1alpha1.LabelSystemManaged,
	}
	roleBindingList, err := roleBindingIfc.List(ctx, listOptions)
	if err != nil {
		err = errors.WithMessagef(err,
			"failed to get all managed RoleBindings from namespace %q",
			namespace,
		)
		return nil, err
	}
	return roleBindingList, nil
}

func (c *Controller) createRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	if c.testing != nil && c.testing.createRoleBindingStub != nil {
		return c.testing.createRoleBindingStub(roleBinding)
	}

	namespace := roleBinding.GetNamespace()
	roleBindingIfc := c.factory.RbacV1().RoleBindings(namespace)
	resultingRoleBinding, err := roleBindingIfc.Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		err = errors.WithMessagef(err,
			"failed to create a RoleBinding in namespace %q",
			namespace,
		)
		return nil, err
	}
	return resultingRoleBinding, nil
}

func (c *Controller) deleteRoleBindingsFromList(ctx context.Context, roleBindingList *rbacv1.RoleBindingList) error {
	for _, roleBinding := range roleBindingList.Items {
		err := c.deleteRoleBinding(ctx, &roleBinding)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) deleteRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding) error {
	if roleBinding.GetName() == "" || roleBinding.GetUID() == "" {
		// object is not uniquely identified
		// treat as if not found
		return nil
	}
	namespace := roleBinding.GetNamespace()
	roleBindingIfc := c.factory.RbacV1().RoleBindings(namespace)
	deleteOptions := metav1.NewDeleteOptions(0)
	deleteOptions.Preconditions = metav1.NewUIDPreconditions(string(roleBinding.GetUID()))
	err := roleBindingIfc.Delete(ctx, roleBinding.GetName(), *deleteOptions)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		err = errors.WithMessagef(err,
			"failed to delete RoleBinding %q in namespace %q",
			roleBinding.GetName(), namespace,
		)
		return err
	}
	return nil
}

func (c *Controller) formatLog(tenant *stewardv1alpha1.Tenant, v ...interface{}) string {
	return fmt.Sprintf(
		"client %q: tenant %q: %s",
		tenant.GetNamespace(), tenant.GetName(),
		fmt.Sprint(v...),
	)
}

func (c *Controller) formatLogf(tenant *stewardv1alpha1.Tenant, format string, v ...interface{}) string {
	return c.formatLog(tenant, fmt.Sprintf(format, v...))
}

func (c *Controller) updateMetrics() {
	// TODO determine number of tenants per client
	list, err := c.tenantLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Cannot update tenant metrics: %s", err.Error())
	}
	count := len(list)
	metrics.TenantCount.Set(float64(count))
}

func (c *Controller) onTenantAdd(obj interface{}) {
	key := c.getKey(obj)
	c.addToQueue(key, "Add")
}

func (c *Controller) onTenantUpdate(old, new interface{}) {
	oldVersion := old.(*stewardv1alpha1.Tenant).GetObjectMeta().GetResourceVersion()
	newVersion := new.(*stewardv1alpha1.Tenant).GetObjectMeta().GetResourceVersion()
	key := c.getKey(new)
	if oldVersion != newVersion {
		//changed
		c.addToQueue(key, "Update (changed)")
		//log.Printf("   diff: %s", cmp.Diff(old, new)) // import github.com/google/go-cmp/cmp
	} else {
		//unchanged - anyway, resync to check if namespace, rolebinding, etc. still exist
		c.addToQueue(key, "Update (unchanged)")
	}
}

func (c *Controller) addToQueue(key string, eventType string) {
	if key == "" {
		klog.V(1).Infof("WARN: '%s' event - key empty, skipping item", eventType)
	} else {
		klog.V(4).Infof("'%s' event - Add to workqueue '%s'", eventType, key)
		c.workqueue.Add(key)
	}
}

func (c *Controller) getKey(obj interface{}) string {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return ""
	}
	return key
}

func (c *Controller) onTenantDelete(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("'Delete' event - could not identify key: %s", err.Error())
	} else {
		klog.V(3).Infof("'Delete' event - '%s'", key)
	}
	c.updateMetrics()
}
