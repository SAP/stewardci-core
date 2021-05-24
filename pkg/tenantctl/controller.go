/*
based on sample-controller from https://github.com/kubernetes/sample-controller/blob/7047ee6ceceef2118a2017bbfff4a86c1f56f1ca/controller.go
*/

package tenantctl

import (
	"fmt"
	"time"

	"github.com/SAP/stewardci-core/pkg/apis/steward"
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	listers "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	slabels "github.com/SAP/stewardci-core/pkg/stewardlabels"
	"github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	knativeapis "knative.dev/pkg/apis"
)

const (
	kind = "Tenants"

	tenantNamespaceRoleBindingNamePrefix = steward.GroupName + "--tenant-role-binding-"
)

// Controller for Steward Tenants
type Controller struct {
	factory      k8s.ClientFactory
	fetcher      k8s.TenantFetcher
	tenantSynced cache.InformerSynced
	tenantLister listers.TenantLister
	workqueue    workqueue.RateLimitingInterface
	metrics      Metrics
	syncCount    int64
	testing      *controllerTesting
}

type controllerTesting struct {
	createRoleBindingStub          func(roleBinding *rbacv1beta1.RoleBinding) (*rbacv1beta1.RoleBinding, error)
	getClientConfigStub            func(factory k8s.ClientFactory, clientNamespace string) (clientConfig, error)
	listManagedRoleBindingsStub    func(namespace string) (*rbacv1beta1.RoleBindingList, error)
	reconcileTenantRoleBindingStub func(tenant *api.Tenant, namespace string, config clientConfig) (bool, error)
	updateStatusStub               func(tenant *api.Tenant) (*api.Tenant, error)
}

// NewController creates new Controller
func NewController(factory k8s.ClientFactory, metrics Metrics) *Controller {
	informer := factory.StewardInformerFactory().Steward().V1alpha1().Tenants()
	fetcher := k8s.NewListerBasedTenantFetcher(informer.Lister())
	controller := &Controller{
		factory:      factory,
		fetcher:      fetcher,
		tenantSynced: informer.Informer().HasSynced,
		tenantLister: informer.Lister(),
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), kind),
		metrics:      metrics,
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

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the tenant resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	origTenant, err := c.fetcher.ByKey(key)
	if err != nil {
		return err
	}

	if origTenant == nil {
		return nil
	}

	tenant := origTenant.DeepCopy()

	klog.V(4).Infof(c.formatLog(tenant, "started reconciliation"))
	if klog.V(4).Enabled() {
		defer klog.V(4).Infof(c.formatLog(&api.Tenant{ObjectMeta: *tenant.ObjectMeta.DeepCopy()}, "finished reconciliation"))
	}

	// the configuration should be loaded once per sync to avoid inconsistencies
	// in case of concurrent configuration changes
	config, err := c.getClientConfig(c.factory, tenant.GetNamespace())
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
		err = c.deleteTenantNamespace(tenant, config)
		if err != nil {
			return err
		}
		_, err = c.removeFinalizerAndUpdate(tenant)
		if err == nil {
			c.syncCount++
		}
		return err
	}

	// Add finalizer if it doesn't exist
	changed, finalizerList := utils.AddStringIfMissing(tenant.GetFinalizers(), k8s.FinalizerName)
	if changed {
		tenant.SetFinalizers(finalizerList)
		_, err:= c.update(tenant)
		return err
	}

	reconcileErr := c.reconcile(config, tenant)

	// do not update the status if there's no change
	if !equality.Semantic.DeepEqual(origTenant.Status, tenant.Status) {
		if _, err := c.updateStatus(tenant); err != nil {
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

func (c *Controller) isInitialized(tenant *api.Tenant) bool {
	return tenant.Status.TenantNamespaceName != ""
}

func (c *Controller) reconcile(config clientConfig, tenant *api.Tenant) (err error) {
	if c.isInitialized(tenant) {
		err = c.reconcileInitialized(config, tenant)
	} else {
		err = c.reconcileUninitialized(config, tenant)
	}
	return
}

func (c *Controller) reconcileUninitialized(config clientConfig, tenant *api.Tenant) error {
	klog.V(3).Infof(c.formatLog(tenant, "tenant not initialized yet"))

	// Check whether namespace is already created for current tenant in case the controller failed update tenant status
	nsName, err := c.getTenantNamespace(config, tenant)
	if err != nil {
		klog.V(4).Info(c.formatLog(tenant), err)
		tenant.Status.SetCondition(&knativeapis.Condition{
			Type:    knativeapis.ConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  api.StatusReasonFailed,
			Message: err.Error(),
		})
		return err
	}

	if nsName == "" {
		if nsName, err = c.createTenantNamespace(config, tenant); err != nil {
			condMsg := "Failed to create a new tenant namespace."
			tenant.Status.SetCondition(&knativeapis.Condition{
				Type:    knativeapis.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  api.StatusReasonFailed,
				Message: condMsg,
			})
			return err
		}
	}

	_, err = c.reconcileTenantRoleBinding(tenant, nsName, config)
	if err != nil {
		condMsg := "Failed to initialize a new tenant namespace because the RoleBinding could not be created."
		tenant.Status.SetCondition(&knativeapis.Condition{
			Type:    knativeapis.ConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  api.StatusReasonFailed,
			Message: condMsg,
		})
		return err
	}

	tenant.Status.TenantNamespaceName = nsName

	tenant.Status.SetCondition(&knativeapis.Condition{
		Type:   knativeapis.ConditionReady,
		Status: corev1.ConditionTrue,
	})

	return nil
}

func (c *Controller) reconcileInitialized(config clientConfig, tenant *api.Tenant) error {
	klog.V(4).Infof(c.formatLog(tenant, "tenant is initialized already"))

	nsName := tenant.Status.TenantNamespaceName

	exists, err := c.checkNamespaceExists(nsName)
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
			Reason:  api.StatusReasonDependentResourceState,
			Message: condMsg,
		})
		err = errors.Errorf("tenant namespace %q does not exist anymore", nsName)
		klog.V(3).Infof(c.formatLog(tenant), err)
		return err
	}

	needForUpdateDetected, err := c.reconcileTenantRoleBinding(tenant, nsName, config)
	if err != nil {
		if needForUpdateDetected {
			condMsg := fmt.Sprintf(
				"The RoleBinding in tenant namespace %q is outdated but could not be updated.",
				nsName,
			)
			tenant.Status.SetCondition(&knativeapis.Condition{
				Type:    knativeapis.ConditionReady,
				Status:  corev1.ConditionFalse,
				Reason:  api.StatusReasonDependentResourceState,
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

func (c *Controller) getClientConfig(factory k8s.ClientFactory, clientNamespace string) (clientConfig, error) {
	if c.testing != nil && c.testing.getClientConfigStub != nil {
		return c.testing.getClientConfigStub(factory, clientNamespace)
	}
	return getClientConfig(factory, clientNamespace)
}

func (c *Controller) hasFinalizer(tenant *api.Tenant) bool {
	return utils.StringSliceContains(tenant.GetFinalizers(), k8s.FinalizerName)
}

func (c *Controller) removeFinalizerAndUpdate(tenant *api.Tenant) (*api.Tenant, error) {
	changed, finalizerList := utils.RemoveString(tenant.GetFinalizers(), k8s.FinalizerName)
	if changed {
		tenant.SetFinalizers(finalizerList)
		return c.update(tenant)
	}
	return tenant, nil
}

func (c *Controller) updateStatus(tenant *api.Tenant) (*api.Tenant, error) {
	if c.testing != nil && c.testing.updateStatusStub != nil {
		return c.testing.updateStatusStub(tenant)
	}

	client := c.factory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	updatedTenant, err := client.UpdateStatus(tenant)
	if err != nil {
		err = errors.WithMessage(err, "failed to update resource status")
		klog.V(3).Infof(c.formatLog(tenant), err)
		return nil, err
	}
	return updatedTenant, nil
}

func (c *Controller) update(tenant *api.Tenant) (*api.Tenant, error) {
	client := c.factory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	result, err := client.Update(tenant)
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

func (c *Controller) checkNamespaceExists(name string) (bool, error) {
	namespaces := c.factory.CoreV1().Namespaces()
	namespace, err := namespaces.Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		err = errors.WithMessagef(err, "error: failed to get namespace %q", name)
		return false, err
	}
	return namespace.GetDeletionTimestamp().IsZero(), nil
}

func (c *Controller) getTenantNamespace(config clientConfig, tenant *api.Tenant) (string, error) {
	klog.V(4).Infof(c.formatLog(tenant, "get namespace for tenant"))
	var nsName string

	namespaceManager := c.getNamespaceManager(config)
	namespaces, err := namespaceManager.List(tenant.GetName())
	if err != nil {
		return "", err
	}

	if len(namespaces) > 1 {
		msg := fmt.Sprintf("found more than one namespaces %v", namespaces)
		err := errors.Errorf(c.formatLog(tenant, msg))
		return "", err
	} else if len(namespaces) == 1 {
		nsName = namespaces[0]
	}

	return nsName, nil
}

func (c *Controller) createTenantNamespace(config clientConfig, tenant *api.Tenant) (string, error) {
	klog.V(4).Infof(c.formatLog(tenant, "creating new tenant namespace"))
	namespaceManager := c.getNamespaceManager(config)
	nsName, err := namespaceManager.Create(tenant.GetName(), nil)
	if err != nil {
		err = errors.WithMessage(err, "failed to create new tenant namespace")
		klog.V(4).Infof(c.formatLog(tenant), err)
		return "", err
	}
	return nsName, err
}

func (c *Controller) deleteTenantNamespace(tenant *api.Tenant, config clientConfig) error {
	klog.V(4).Infof(c.formatLogf(tenant, "rolling back tenant namespace"))
	ns, err := c.getTenantNamespace(config, tenant)
	if err != nil {
		err = errors.WithMessagef(err, "failed to delete tenant namespace")
		klog.V(4).Infof(c.formatLog(tenant), err)
		return err
	}

	if ns == "" {
		return nil
	}

	namespaceManager := c.getNamespaceManager(config)
	err = namespaceManager.Delete(ns)
	if err != nil {
		err = errors.WithMessagef(err, "failed to delete tenant namespace %q", ns)
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
func (c *Controller) reconcileTenantRoleBinding(tenant *api.Tenant, namespace string, config clientConfig) (needForUpdateDetected bool, err error) {
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
		rbList, err := c.listManagedRoleBindings(namespace)
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
			_, err = c.createRoleBinding(expectedTenantRB)
			if err != nil {
				return err
			}
			err = c.deleteRoleBindingsFromList(rbList)
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
) *rbacv1beta1.RoleBinding {
	roleBinding := &rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			// let the server generate a unique name
			GenerateName: tenantNamespaceRoleBindingNamePrefix,
			Namespace:    tenantNamespace,
		},
		RoleRef: rbacv1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     string(config.GetTenantRoleName()),
		},
		Subjects: []rbacv1beta1.Subject{
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

func (c *Controller) isTenantRoleBindingUpToDate(current *rbacv1beta1.RoleBinding, expected *rbacv1beta1.RoleBinding) bool {
	return true &&
		equality.Semantic.DeepEqual(expected.GetLabels(), current.GetLabels()) &&
		equality.Semantic.DeepEqual(expected.GetAnnotations(), current.GetAnnotations()) &&
		equality.Semantic.DeepEqual(expected.RoleRef, current.RoleRef) &&
		equality.Semantic.DeepEqual(expected.Subjects, current.Subjects)
}

func (c *Controller) listManagedRoleBindings(namespace string) (*rbacv1beta1.RoleBindingList, error) {
	if c.testing != nil && c.testing.listManagedRoleBindingsStub != nil {
		return c.testing.listManagedRoleBindingsStub(namespace)
	}

	roleBindingIfc := c.factory.RbacV1beta1().RoleBindings(namespace)
	listOptions := metav1.ListOptions{
		LabelSelector: api.LabelSystemManaged,
	}
	roleBindingList, err := roleBindingIfc.List(listOptions)
	if err != nil {
		err = errors.WithMessagef(err,
			"failed to get all managed RoleBindings from namespace %q",
			namespace,
		)
		return nil, err
	}
	return roleBindingList, nil
}

func (c *Controller) createRoleBinding(roleBinding *rbacv1beta1.RoleBinding) (*rbacv1beta1.RoleBinding, error) {
	if c.testing != nil && c.testing.createRoleBindingStub != nil {
		return c.testing.createRoleBindingStub(roleBinding)
	}

	namespace := roleBinding.GetNamespace()
	roleBindingIfc := c.factory.RbacV1beta1().RoleBindings(namespace)
	resultingRoleBinding, err := roleBindingIfc.Create(roleBinding)
	if err != nil {
		err = errors.WithMessagef(err,
			"failed to create a RoleBinding in namespace %q",
			namespace,
		)
		return nil, err
	}
	return resultingRoleBinding, nil
}

func (c *Controller) deleteRoleBindingsFromList(roleBindingList *rbacv1beta1.RoleBindingList) error {
	for _, roleBinding := range roleBindingList.Items {
		err := c.deleteRoleBinding(&roleBinding)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) deleteRoleBinding(roleBinding *rbacv1beta1.RoleBinding) error {
	if roleBinding.GetName() == "" || roleBinding.GetUID() == "" {
		// object is not uniquely identified
		// treat as if not found
		return nil
	}
	namespace := roleBinding.GetNamespace()
	roleBindingIfc := c.factory.RbacV1beta1().RoleBindings(namespace)
	deleteOptions := metav1.NewDeleteOptions(0)
	deleteOptions.Preconditions = metav1.NewUIDPreconditions(string(roleBinding.GetUID()))
	err := roleBindingIfc.Delete(roleBinding.GetName(), deleteOptions)
	if k8serrors.IsNotFound(err) {
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

func (c *Controller) formatLog(tenant *api.Tenant, v ...interface{}) string {
	return fmt.Sprintf(
		"client %q: tenant %q: %s",
		tenant.GetNamespace(), tenant.GetName(),
		fmt.Sprint(v...),
	)
}

func (c *Controller) formatLogf(tenant *api.Tenant, format string, v ...interface{}) string {
	return c.formatLog(tenant, fmt.Sprintf(format, v...))
}

func (c *Controller) updateMetrics() {
	// TODO determine number of tenants per client
	list, err := c.tenantLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Cannot update tenant metrics: %s", err.Error())
	}
	count := len(list)
	c.metrics.SetTenantNumber(float64(count))
}

func (c *Controller) onTenantAdd(obj interface{}) {
	key := c.getKey(obj)
	c.addToQueue(key, "Add")
}

func (c *Controller) onTenantUpdate(old, new interface{}) {
	oldVersion := old.(*api.Tenant).GetObjectMeta().GetResourceVersion()
	newVersion := new.(*api.Tenant).GetObjectMeta().GetResourceVersion()
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
