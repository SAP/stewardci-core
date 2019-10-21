/*
based on sample-controller from https://github.com/kubernetes/sample-controller/blob/7047ee6ceceef2118a2017bbfff4a86c1f56f1ca/controller.go
*/

package tenantctl

import (
	"fmt"
	"log"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	listers "github.com/SAP/stewardci-core/pkg/client/listers/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	utils "github.com/SAP/stewardci-core/pkg/utils"
	"github.com/pkg/errors"
	v1beta1 "k8s.io/api/rbac/v1beta1"
	labels "k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	wait "k8s.io/apimachinery/pkg/util/wait"
	cache "k8s.io/client-go/tools/cache"
	workqueue "k8s.io/client-go/util/workqueue"
)

const kind = "Tenants"
const defaultServiceAccountName = "default"

// Controller for Steward
type Controller struct {
	factory      k8s.ClientFactory
	fetcher      k8s.TenantFetcher
	tenantSynced cache.InformerSynced
	tenantLister listers.TenantLister
	workqueue    workqueue.RateLimitingInterface
	metrics      Metrics
	syncCount    int64
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
		AddFunc:    controller.addTenant,
		UpdateFunc: controller.updateTenant,
		DeleteFunc: controller.deleteTenant,
	})
	return controller
}

func (c *Controller) getSyncCount() int64 {
	return c.syncCount
}

func (c *Controller) getNamespaceManager(tenant *api.Tenant) (k8s.NamespaceManager, error) {
	config, err := getClientConfig(c.factory, tenant.GetNamespace())
	if err != nil {
		return nil, err
	}
	tenantNamespacePrefix := config.GetTenantNamespacePrefix()
	namespaceManager := k8s.NewNamespaceManager(c.factory, tenantNamespacePrefix,
		config.GetTenantNamespaceSuffixLength())
	return namespaceManager, nil
}

// Run runs the controller
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
	tenant, err := c.fetcher.ByKey(key)
	if err != nil {
		return err
	}

	if tenant == nil {
		return nil
	}

	// Check if object has deletion timestamp
	// If not, try to add finalizer if missing
	if tenant.ObjectMeta.DeletionTimestamp.IsZero() {
		changed, finalizerList := utils.AddStringIfMissing(tenant.ObjectMeta.Finalizers, k8s.FinalizerName)
		if changed {
			tenant.ObjectMeta.Finalizers = finalizerList
			_, err = c.update(tenant)
			return err
		}
	} else {
		err := c.rollback(tenant)
		if err != nil {
			log.Printf("ERROR: Deletion of NS %s failed: %v", tenant.Status.TenantNamespaceName, err.Error())
			return err
		}
		err = c.removeFinalizer(tenant)
		if err == nil {
			c.syncCount++
		}
		return err
	}

	if tenant.Status.Progress != api.TenantProgressUndefined && tenant.Status.Progress != api.TenantProgressFinished {
		//TODO: We need to handle this resiliently, not exit
		err := fmt.Errorf("Tenant '%s' in namespace '%s' seems to have failed previously in step '%s'", tenant.GetName(), tenant.GetNamespace(), tenant.Status.Progress)
		log.Printf("ERROR: %s", err.Error())
		return nil //err <- TODO: as long as we do not fix the error state it does not make sense to retry
	}

	defer c.rollbackIfRequired(key)

	// Check if tenant setup is completed
	if tenant.Status.Progress != api.TenantProgressFinished {
		tenant, _ = c.updateProgress(tenant, api.TenantProgressInProcess)
		var err error
		var namespaceName string
		var account *k8s.ServiceAccountWrap

		config, err := getClientConfig(c.factory, tenant.GetNamespace())
		if err != nil {
			log.Printf("ERROR: Could not get config: %s", err.Error())
			return err
		}
		tenantRoleName := config.GetTenantRoleName()

		//TODO: handle updateProgress errors
		tenant, _ = c.updateProgress(tenant, api.TenantProgressCreateNamespace)
		namespaceName, err = c.createNamespace(tenant)
		if err != nil {
			return c.handleError(tenant, err, api.TenantResultErrorContent)
		}
		log.Printf("Create namespace successful for %s", namespaceName)

		tenant, _ = c.updateProgress(tenant, api.TenantProgressGetServiceAccount)
		account, err = c.getServiceAccount(tenant, defaultServiceAccountName)
		if err != nil {
			return c.handleError(tenant, err, api.TenantResultErrorInfra)
		}

		tenant, _ = c.updateProgress(tenant, api.TenantProgressAddRoleBinding)
		var roleBinding *v1beta1.RoleBinding
		roleBinding, err = c.addRoleBinding(account, tenant, tenantRoleName)
		if err != nil {
			return c.handleError(tenant, err, api.TenantResultErrorInfra)
		}
		log.Printf("Created Role Binding '%s' in namespace '%s'", roleBinding.GetName(), namespaceName)

		tenant, _ = c.updateProgress(tenant, api.TenantProgressFinalize)
		tenant.Status.Result = api.TenantResultSuccess
		tenant.Status.Message = "Tenant namespace successfully prepared"
		if tenant, err = c.updateStatus(tenant); err != nil {
			return err
		}
		tenant, _ = c.updateProgress(tenant, api.TenantProgressFinished)
		log.Printf("Tenant preparation successful for %s", tenant.GetName())
	}
	c.updateMetrics()
	c.syncCount++
	return nil
}

func (c *Controller) removeFinalizer(tenant *api.Tenant) error {
	changed, finalizerList := utils.RemoveString(tenant.ObjectMeta.Finalizers, k8s.FinalizerName)
	if changed {
		tenant.ObjectMeta.Finalizers = finalizerList
		_, err := c.update(tenant)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) updateProgress(tenant *api.Tenant, progress api.TenantCreationProgress) (*api.Tenant, error) {
	tenant.Status.Progress = progress
	return c.updateStatus(tenant)
}

func (c *Controller) updateStatus(tenant *api.Tenant) (*api.Tenant, error) {
	client := c.factory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	updatedTenant, err := client.UpdateStatus(tenant)
	if err != nil {
		err = errors.WithMessagef(err, "Failed to update status of tenant '%s' in namespace '%s'", tenant.GetName(), tenant.GetNamespace())
		log.Printf("ERROR: %s", err.Error())
		return nil, err
	}
	return updatedTenant, nil
}

func (c *Controller) update(tenant *api.Tenant) (*api.Tenant, error) {
	client := c.factory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	updatedTenant, err := client.Update(tenant)
	if err != nil {
		err = errors.WithMessagef(err, "Failed to update tenant '%s' in namespace '%s'", tenant.GetName(), tenant.GetNamespace())
		log.Printf("ERROR: %s", err.Error())
		return nil, err
	}
	return updatedTenant, nil
}

// An error is returned in cases to signalize processNextWorkItem() to retry processing the tenant.
// If no error is returned this signalized OK, do not retry. This should be done in cases where retry will not help.
func (c *Controller) handleError(tenant *api.Tenant, err error, result api.TenantResult) error {
	log.Printf("ERROR: %s", err.Error())
	tenant.Status.Result = result
	tenant.Status.Message = utils.Trim(err.Error())
	_, updateStatusErr := c.updateStatus(tenant)
	return updateStatusErr
}

func (c *Controller) rollbackIfRequired(tenantKey string) {
	tenant, err := c.fetcher.ByKey(tenantKey)
	if err != nil {
		log.Printf("ERROR: Could not get tenant during rollback: %s", err.Error())
	}
	if tenant.Status.Progress != api.TenantProgressFinished {
		_ = c.rollback(tenant)
	}
}

func (c *Controller) rollback(tenant *api.Tenant) error {
	log.Printf("Rollback tenant %s", tenant.GetName())
	if tenant.Status.TenantNamespaceName == "" {
		log.Printf("Nothing to rollback for tenant %s", tenant.GetName())
	} else {
		err := c.deleteNamespace(tenant)
		if err != nil {
			log.Printf("ERROR: Deletion of %s failed: %v", tenant.Status.TenantNamespaceName, err.Error())
			return err
		}
	}
	return nil
}

func (c *Controller) deleteNamespace(tenant *api.Tenant) error {
	namespaceManager, err := c.getNamespaceManager(tenant)
	if err != nil {
		err = errors.WithMessage(err, "Could not delete namespace")
		return err
	}
	return namespaceManager.Delete(tenant.Status.TenantNamespaceName)
}

func (c *Controller) createNamespace(tenant *api.Tenant) (string, error) {
	log.Printf("Create namespace for: %s", tenant.GetName())
	annotations := map[string]string{}
	namespaceManager, err := c.getNamespaceManager(tenant)
	if err != nil {
		err = errors.WithMessage(err, "Could not get namespace manager")
		return "", err
	}

	fullName, err := namespaceManager.Create(tenant.GetName(), annotations)
	if err == nil {
		tenant.Status.TenantNamespaceName = fullName
	} else {
		err = errors.WithMessagef(err, "Create namespace failed for tenant %s:", tenant.GetName())
	}
	return fullName, err
}

func (c *Controller) getServiceAccount(tenant *api.Tenant, serviceAccountName string) (*k8s.ServiceAccountWrap, error) {
	log.Printf("Get service account %s for: %s", serviceAccountName, tenant.GetName())
	accountManager := k8s.NewServiceAccountManager(c.factory, tenant.GetNamespace())
	account, err := accountManager.GetServiceAccount(serviceAccountName)
	if err != nil {
		err = errors.WithMessagef(err, "Fetch service account failed for %s", tenant.Status.TenantNamespaceName)
	}
	return account, err
}

func (c *Controller) addRoleBinding(account *k8s.ServiceAccountWrap, tenant *api.Tenant, role k8s.RoleName) (*v1beta1.RoleBinding, error) {
	log.Printf("Add role binding to role %s in namespace %s", role, tenant.Status.TenantNamespaceName)
	roleBinding, err := account.AddRoleBinding(role, tenant.Status.TenantNamespaceName)
	if err != nil {
		err = errors.WithMessagef(err, "Add Role Binding to service account failed for %s", tenant.Status.TenantNamespaceName)
		tenant.Status.Result = api.TenantResultErrorInfra
		tenant.Status.Message = utils.Trim(err.Error())
		log.Printf("ERROR: %s", err.Error())
	}
	return roleBinding, err
}

func (c *Controller) updateMetrics() {
	list, err := c.tenantLister.List(labels.Everything())
	if err != nil {
		log.Printf("Cannot update tenant metrics: %s", err.Error())
	}
	count := len(list)
	c.metrics.SetTenantNumber(float64(count))
}

func (c *Controller) addTenant(obj interface{}) {
	key := c.getKey(obj)
	c.addToQueue(key, "Add")
}

func (c *Controller) updateTenant(old, new interface{}) {
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

func (c *Controller) deleteTenant(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("'Delete' event - could not identify key: %s", err.Error())
	} else {
		log.Printf("'Delete' event - '%s'", key)
	}
	c.updateMetrics()
}
