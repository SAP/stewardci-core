/*
based on sample-controller from https://github.com/kubernetes/sample-controller/blob/7047ee6ceceef2118a2017bbfff4a86c1f56f1ca/controller.go
*/

package tenantctl

import (
	"fmt"
	"log"
	"time"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward"
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	listers "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	utils "github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
	cache "k8s.io/client-go/tools/cache"
	workqueue "k8s.io/client-go/util/workqueue"
	knativeapis "knative.dev/pkg/apis"
)

const (
	kind                           = "Tenants"
	tenantNamespaceRoleBindingName = steward.GroupName + "--tenant-role-binding"
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
	getClientConfigStub func(factory k8s.ClientFactory, clientNamespace string) (clientConfig, error)
	syncRoleBindingStub func(tenant *api.Tenant, namespace string, config clientConfig) (bool, error)
	updateStatusStub    func(tenant *api.Tenant) (*api.Tenant, error)
}

// NewController creates new Controller
func NewController(factory k8s.ClientFactory, fetcher k8s.TenantFetcher, metrics Metrics) *Controller {
	informer := factory.StewardInformerFactory().Steward().V1alpha1().Tenants()
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
	log.Printf("Sync cache")
	if ok := cache.WaitForCacheSync(stopCh, c.tenantSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	log.Printf("Start workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	log.Printf("Workers running [%v]", threadiness)
	<-stopCh
	log.Printf("Workers stopped")
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
		log.Printf("Requeued %v times '%s'", numRequeues, obj.(string))
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
		log.Printf("Finished syncing '%s'", key)
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

	c.logPrintln(tenant, "started reconciliation")
	defer c.logPrintln(&api.Tenant{ObjectMeta: *tenant.ObjectMeta.DeepCopy()}, "finished reconciliation")

	// the configuration should be loaded once per sync to avoid inconsistencies
	// in case of concurrent configuration changes
	config, err := c.getClientConfig(c.factory, tenant.GetNamespace())
	if err != nil {
		c.logPrintln(tenant, err)
		return err
	}

	if !tenant.ObjectMeta.DeletionTimestamp.IsZero() {
		c.logPrintln(tenant, "tenant is marked as deleted")
		if !c.hasFinalizer(tenant) {
			c.logPrintln(tenant, "dependent resources cleaned already, nothing to do")
			return nil
		}
		err = c.rollbackTenantNamespace(tenant.Status.TenantNamespaceName, tenant, config)
		if err != nil {
			return err
		}
		tenant, err = c.removeFinalizerAndUpdate(tenant)
		if err == nil {
			c.syncCount++
		}
		return err
	}

	tenant, err = c.addFinalizerAndUpdate(tenant)
	if err != nil {
		return err
	}

	reconcileErr := c.reconcile(config, tenant)

	// do not update the status if there's no change
	if !equality.Semantic.DeepEqual(origTenant.Status, tenant.Status) {
		if _, err := c.updateStatus(tenant); err != nil {
			if !c.isInitialized(origTenant) && c.isInitialized(tenant) {
				c.rollbackTenantNamespace(tenant.Status.TenantNamespaceName, tenant, config)
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
	c.logPrintln(tenant, "tenant not initialized yet")

	nsName, err := c.createTenantNamespace(config, tenant)
	if err != nil {
		condMsg := fmt.Sprintf("Failed to create the tenant namespace.")
		tenant.Status.SetCondition(&knativeapis.Condition{
			Type:    knativeapis.ConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  api.StatusReasonFailed,
			Message: condMsg,
		})
		return err
	}

	_, err = c.syncRoleBinding(tenant, nsName, config)
	if err != nil {
		condMsg := fmt.Sprintf("Failed to create the tenant namespace.")
		tenant.Status.SetCondition(&knativeapis.Condition{
			Type:    knativeapis.ConditionReady,
			Status:  corev1.ConditionFalse,
			Reason:  api.StatusReasonFailed,
			Message: condMsg,
		})
		c.rollbackTenantNamespace(nsName, tenant, config) // clean-up ignoring error
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
	c.logPrintln(tenant, "tenant is initialized already")

	nsName := tenant.Status.TenantNamespaceName

	exists, err := c.checkNamespaceExists(nsName)
	if err != nil {
		c.logPrintln(tenant, err)
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
		c.logPrintln(tenant, err)
		return err
	}

	syncNeeded, err := c.syncRoleBinding(tenant, nsName, config)
	if err != nil {
		if syncNeeded {
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

func (c *Controller) addFinalizerAndUpdate(tenant *api.Tenant) (*api.Tenant, error) {
	changed, finalizerList := utils.AddStringIfMissing(tenant.GetFinalizers(), k8s.FinalizerName)
	if changed {
		tenant.SetFinalizers(finalizerList)
		return c.update(tenant)
	}
	return tenant, nil
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
		c.logPrintln(tenant, err)
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
		c.logPrintln(tenant, err)
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

func (c *Controller) createTenantNamespace(config clientConfig, tenant *api.Tenant) (string, error) {
	c.logPrintln(tenant, "creating new tenant namespace")
	namespaceManager := c.getNamespaceManager(config)
	nsName, err := namespaceManager.Create(tenant.GetName(), nil)
	if err != nil {
		err = errors.WithMessage(err, "failed to create new tenant namespace")
		c.logPrintln(tenant, err)
		return "", err
	}
	return nsName, err
}

func (c *Controller) rollbackTenantNamespace(namespace string, tenant *api.Tenant, config clientConfig) error {
	if namespace == "" {
		return nil
	}
	c.logPrintf(tenant, "rolling back tenant namespace %q", namespace)
	namespaceManager := c.getNamespaceManager(config)
	err := namespaceManager.Delete(namespace)
	if err != nil {
		err = errors.WithMessagef(err, "failed to rollback tenant namespace %q", namespace)
		c.logPrintln(tenant, err)
		return err
	}
	return nil
}

func (c *Controller) syncRoleBinding(tenant *api.Tenant, namespace string, config clientConfig) (bool, error) {
	if c.testing != nil && c.testing.syncRoleBindingStub != nil {
		return c.testing.syncRoleBindingStub(tenant, namespace, config)
	}

	syncNeeded, err := func() (bool, error) {
		rbName := tenantNamespaceRoleBindingName
		current, err := c.getRoleBinding(rbName, namespace)
		if err != nil {
			return false, err
		}
		clientNamespace := tenant.GetNamespace()
		expected := c.generateRoleBinding(rbName, namespace, clientNamespace, config)
		if current == nil ||
			!equality.Semantic.DeepEqual(expected.GetLabels(), current.GetLabels()) ||
			!equality.Semantic.DeepEqual(expected.GetAnnotations(), current.GetAnnotations()) ||
			!equality.Semantic.DeepEqual(expected.RoleRef, current.RoleRef) ||
			!equality.Semantic.DeepEqual(expected.Subjects, current.Subjects) {

			_, err = c.createOrReplaceRoleBinding(expected)
			if err != nil {
				return true, err
			}
			return true, nil
		}
		return false, nil
	}()

	if err != nil {
		err = errors.WithMessagef(err,
			"failed to sync the RoleBinding in tenant namespace %q",
			namespace,
		)
		c.logPrintln(tenant, err)
	}
	return syncNeeded, err
}

/**
 * generateRoleBinding generates the role binding for a tenant namespace
 * as in-memory object only (no persistence in K8s).
 */
func (c *Controller) generateRoleBinding(
	name string, tenantNamespace string, clientNamespace string, config clientConfig,
) *rbacv1beta1.RoleBinding {
	return &rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: tenantNamespace,
			Labels: map[string]string{
				api.LabelSystemManaged: "",
			},
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
}

func (c *Controller) getRoleBinding(name string, namespace string) (*rbacv1beta1.RoleBinding, error) {
	roleBindings := c.factory.RbacV1beta1().RoleBindings(namespace)
	roleBinding, err := roleBindings.Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		err = errors.WithMessagef(err,
			"failed to get RoleBinding %q in namespace %q",
			name, namespace,
		)
		return nil, err
	}
	return roleBinding, nil
}

func (c *Controller) createOrReplaceRoleBinding(roleBinding *rbacv1beta1.RoleBinding) (*rbacv1beta1.RoleBinding, error) {
	name := roleBinding.GetName()
	namespace := roleBinding.GetNamespace()
	roleBindings := c.factory.RbacV1beta1().RoleBindings(roleBinding.GetNamespace())
	resultingRoleBinding, err := roleBindings.Create(roleBinding)
	if k8serrors.IsAlreadyExists(err) {
		resultingRoleBinding, err = roleBindings.Update(roleBinding)
	}
	if err != nil {
		err = errors.WithMessagef(err,
			"failed to create/replace RoleBinding %q in namespace %q",
			name, namespace,
		)
		return nil, err
	}
	return resultingRoleBinding, nil
}

func (c *Controller) logPrintln(tenant *api.Tenant, v ...interface{}) {
	log.Printf(
		"client %q: tenant %q: %s",
		tenant.GetNamespace(), tenant.GetName(),
		fmt.Sprint(v...),
	)
}

func (c *Controller) logPrintf(tenant *api.Tenant, format string, v ...interface{}) {
	c.logPrintln(tenant, fmt.Sprintf(format, v...))
}

func (c *Controller) updateMetrics() {
	// TODO determine number of tenants per client
	list, err := c.tenantLister.List(labels.Everything())
	if err != nil {
		log.Printf("Cannot update tenant metrics: %s", err.Error())
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
		log.Printf("WARN: '%s' event - key empty, skipping item", eventType)
	} else {
		log.Printf("'%s' event - Add to workqueue '%s'", eventType, key)
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
		log.Printf("'Delete' event - could not identify key: %s", err.Error())
	} else {
		log.Printf("'Delete' event - '%s'", key)
	}
	c.updateMetrics()
}
