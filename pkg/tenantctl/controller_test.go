package tenantctl

import (
	"fmt"
	"strings"
	"testing"
	"time"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	assert "gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ns1 = "clientNamespace1"

var (
	optList               = metav1.ListOptions{}
	optGet                = metav1.GetOptions{}
	optDelete             = &metav1.DeleteOptions{}
	defaultServiceAccount = &v1.ServiceAccount{ObjectMeta: fake.ObjectMeta("default", ns1)}
	defaultTenantRoleName = "testrole"
)

const (
	tenantID1 = "tenantID1"
	tenantID2 = "tenantID2"
	tenantID3 = "tenantID3"
)

const prefix1 = "prefix1"

func Test_Controller(t *testing.T) {
	cf := fake.NewClientFactory(
		fake.NamespaceWithAnnotations(ns1, map[string]string{
			steward.AnnotationTenantNamespacePrefix: prefix1,
			steward.AnnotationTenantRole:            defaultTenantRoleName,
		}),
		defaultServiceAccount,
		fake.ClusterRole(defaultTenantRoleName),
		fake.Tenant(tenantID1, "TenantName", "Description", ns1),
	)

	stopCh, _ := startController(t, cf)
	defer stopController(t, stopCh)

	assertTenant(t, cf, ns1, tenantID1, expect{
		name:                "TenantName",
		displayName:         "Description",
		result:              steward.TenantResultSuccess,
		message:             "Tenant namespace successfully prepared",
		prefix:              prefix1,
		namespaceExists:     true,
		namespaceStartsWith: prefix1 + "-" + tenantID1,
	})
}

func Test_MultipleTenants(t *testing.T) {
	cf := fake.NewClientFactory(
		fake.NamespaceWithAnnotations(ns1, map[string]string{
			steward.AnnotationTenantNamespacePrefix: prefix1,
			steward.AnnotationTenantRole:            defaultTenantRoleName,
		}),
		defaultServiceAccount,
		fake.ClusterRole(defaultTenantRoleName),
		fake.Tenant(tenantID1, "TenantName1", "Description1", ns1),
		fake.Tenant(tenantID2, "TenantName2", "Description2", ns1),
		fake.Tenant(tenantID3, "TenantName3", "Description3", ns1),
	)

	stopCh, _ := startController(t, cf)
	defer stopController(t, stopCh)

	assertTenant(t, cf, ns1, tenantID1, expect{
		name:                "TenantName1",
		displayName:         "Description1",
		result:              steward.TenantResultSuccess,
		message:             "Tenant namespace successfully prepared",
		prefix:              prefix1,
		namespaceExists:     true,
		namespaceStartsWith: prefix1 + "-" + tenantID1,
	})
	assertTenant(t, cf, ns1, tenantID2, expect{
		name:                "TenantName2",
		displayName:         "Description2",
		result:              steward.TenantResultSuccess,
		message:             "Tenant namespace successfully prepared",
		prefix:              prefix1,
		namespaceExists:     true,
		namespaceStartsWith: prefix1 + "-" + tenantID2,
	})
	assertTenant(t, cf, ns1, tenantID3, expect{
		name:                "TenantName3",
		displayName:         "Description3",
		result:              steward.TenantResultSuccess,
		message:             "Tenant namespace successfully prepared",
		prefix:              prefix1,
		namespaceExists:     true,
		namespaceStartsWith: prefix1 + "-" + tenantID3,
	})
}

func Test_MissingServiceAccount(t *testing.T) {
	cf := fake.NewClientFactory(
		fake.NamespaceWithAnnotations(ns1, map[string]string{
			steward.AnnotationTenantNamespacePrefix: prefix1,
			steward.AnnotationTenantRole:            defaultTenantRoleName,
		}),
		fake.Tenant(tenantID1, "TenantName", "Description", ns1),
	)

	stopCh, _ := startController(t, cf)
	defer stopController(t, stopCh)

	assertTenant(t, cf, ns1, tenantID1, expect{
		name:            "TenantName",
		displayName:     "Description",
		result:          steward.TenantResultErrorInfra,
		message:         `serviceaccounts "` + defaultServiceAccountName + `" not found`,
		prefix:          prefix1,
		namespaceExists: false,
	})
}

func Test_DuplicateTenantID(t *testing.T) {
	clashNamespace := fmt.Sprintf("%s-%s", prefix1, tenantID1)
	cf := fake.NewClientFactory(
		defaultServiceAccount,
		fake.ClusterRole(defaultTenantRoleName),
		fake.NamespaceWithAnnotations(ns1, map[string]string{
			steward.AnnotationTenantNamespacePrefix:       prefix1,
			steward.AnnotationTenantRole:                  defaultTenantRoleName,
			steward.AnnotationTenantNamespaceSuffixLength: "0",
		}),
		fake.Namespace(clashNamespace),
		fake.Tenant(tenantID1, "Duplicate Tenant", "Description", ns1),
	)

	stopCh, _ := startController(t, cf)
	defer stopController(t, stopCh)

	assertTenant(t, cf, ns1, tenantID1, expect{
		name:                "Duplicate Tenant",
		displayName:         "Description",
		result:              steward.TenantResultErrorContent,
		message:             "already exists",
		prefix:              prefix1,
		namespaceStartsWith: "",
		namespaceExists:     false,
	})
	/*TODO: Check if initial conflicting namespace still exists. Unfortunately the fakes have an issue here and
	the create namespace of the controller produced an error 'more than one object matched' but still added the duplicate namespace.
	List namespaces contains both namespaces with the same name. */
	//assertNamespace(t, cf, clashNamespace, true, "")
}

func Test_MissingClusterRole(t *testing.T) {
	cf := fake.NewClientFactory(
		defaultServiceAccount,
		fake.NamespaceWithAnnotations(ns1, map[string]string{
			steward.AnnotationTenantNamespacePrefix:       prefix1,
			steward.AnnotationTenantRole:                  defaultTenantRoleName,
			steward.AnnotationTenantNamespaceSuffixLength: "0",
		}),
		fake.Tenant(tenantID1, "TenantName", "Description", ns1),
	)

	stopCh, _ := startController(t, cf)
	defer stopController(t, stopCh)

	assertTenant(t, cf, ns1, tenantID1, expect{
		name:            "TenantName",
		displayName:     "Description",
		result:          steward.TenantResultErrorInfra,
		message:         `clusterroles.rbac.authorization.k8s.io "` + defaultTenantRoleName + `" not found`,
		prefix:          prefix1,
		namespaceExists: false,
	})
}

//Test for ERROR: Failed to update status of tenant '4e93d9d5-276e-47ca-a570-b3a763aaef3e' in namespace 'stu':
//         Operation cannot be fulfilled on tenants.steward.sap.com "4e93d9d5-276e-47ca-a570-b3a763aaef3e":
//         the object has been modified; please apply your changes to the latest version and try again
func Test_MultipleUpdateStatusCalls(t *testing.T) {
	cf := fake.NewClientFactory(
		fakeClusterRole(),
		fakeServiceAccount(),
		fake.NamespaceWithAnnotations(ns1, map[string]string{
			steward.AnnotationTenantNamespacePrefix: prefix1,
			steward.AnnotationTenantRole:            defaultTenantRoleName,
		}),
		fake.Tenant(tenantID1, "TenantName", "Description", ns1),
	)

	stopCh, controller := startController(t, cf)
	defer stopController(t, stopCh)

	tenant, err := controller.fetcher.ByKey(tenantKey(ns1, tenantID1))
	assert.Equal(t, tenantID1, tenant.GetName())
	tenant.Status.Message = "Changed 1"
	_, err = controller.updateStatus(tenant)
	assert.NilError(t, err)
	tenant.Status.Result = steward.TenantResultErrorInfra
	//TODO: This one here should fail since the original object was updated and not the returned one - but it doesn't with the fakes.
	_, err = controller.updateStatus(tenant)
	//assert.Assert(t, err != nil)
}

func TestFullWorkflow(t *testing.T) {

	const clientNamespace = "client1"
	const defaultTenantID = "tenant1"

	cf := fake.NewClientFactory(
		fake.NamespaceWithAnnotations(clientNamespace, map[string]string{
			steward.AnnotationTenantNamespacePrefix: prefix1,
			steward.AnnotationTenantRole:            defaultTenantRoleName,
		}),
		fake.ServiceAccount(defaultServiceAccountName, clientNamespace),
		fake.ClusterRole(defaultTenantRoleName),
	)

	//Start controller with initially existing K8s resources
	stopCh, controller := startController(t, cf)
	defer stopController(t, stopCh)

	tenantsClient := cf.StewardV1alpha1().Tenants(clientNamespace)
	namespacesClient := cf.CoreV1().Namespaces()

	t.Log("... CHECKS BEFORE")

	tenants, _ := tenantsClient.List(optList)
	assert.Equal(t, 0, len(tenants.Items))

	namespaces, _ := namespacesClient.List(optList)
	assert.Equal(t, 1, len(namespaces.Items))

	_, err := tenantsClient.Get(defaultTenantID, optGet)
	assert.Equal(t, `tenants.steward.sap.com "tenant1" not found`, err.Error())

	clusterRole, err := cf.RbacV1beta1().ClusterRoles().Get(defaultTenantRoleName, optGet)
	assert.NilError(t, err)
	assert.Assert(t, clusterRole != nil)

	serviceAccount, err := cf.CoreV1().ServiceAccounts(clientNamespace).Get(defaultServiceAccountName, optGet)
	assert.NilError(t, err)
	assert.Assert(t, serviceAccount != nil)

	t.Log("...CREATE TENANT")

	syncCount := controller.getSyncCount()
	_, err = tenantsClient.Create(fake.Tenant(defaultTenantID, "name", "dname", clientNamespace))
	assert.NilError(t, err)

	for controller.getSyncCount() <= syncCount {
		sleep(t, 100, "waiting for tenant controller sync")
	}

	t.Log("...CHECKS AFTER CREATE")

	tenants, _ = tenantsClient.List(optList)
	assert.Equal(t, 1, len(tenants.Items))

	namespaces, _ = namespacesClient.List(optList)
	// Client and Tenant Namespace
	assert.Equal(t, 2, len(namespaces.Items))

	tenant, err := tenantsClient.Get(defaultTenantID, optGet)
	assert.NilError(t, err)
	assert.Equal(t, "Tenant namespace successfully prepared", tenant.Status.Message)
	assert.Equal(t, steward.TenantResultSuccess, tenant.Status.Result)

	tenantNamespace := tenant.Status.TenantNamespaceName
	namespace, err := namespacesClient.Get(tenantNamespace, optGet)
	assert.NilError(t, err)
	assert.Assert(t, namespace != nil)

	//Service Account in client namespace is used, not created in tenant namespace
	serviceAccount, err = cf.CoreV1().ServiceAccounts(tenantNamespace).Get(defaultServiceAccountName, optGet)
	assert.Equal(t, `serviceaccounts "`+defaultServiceAccountName+`" not found`, err.Error())

	roleBinding, err := cf.RbacV1beta1().RoleBindings(tenantNamespace).Get(defaultTenantRoleName, optGet)
	assert.NilError(t, err)
	assert.Assert(t, roleBinding != nil)

	t.Log("... CHECK BEFORE DELETION")
	assert.Equal(t, 1, len(tenant.GetFinalizers()))

	t.Log("... DELETE TENANT")

	syncCount = controller.getSyncCount()

	// Fake client immediately deletes client
	// err = tenantsClient.Delete(tenant.GetName(), optDelete)
	// Set deletion timestamp instead
	now := metav1.Now()
	tenant.SetDeletionTimestamp(&now)
	_, err = tenantsClient.Update(tenant)

	assert.NilError(t, err)

	for controller.getSyncCount() <= syncCount {
		sleep(t, 100, "waiting for tenant controller sync")
	}

	t.Log("... CHECKS AFTER DELETE")

	tenant, _ = tenantsClient.Get(tenant.GetName(), optGet)
	// Consider tenant as deleted if deletion timestamp is set and finalizer list is empty
	assert.Assert(t, !tenant.GetDeletionTimestamp().IsZero())
	assert.Equal(t, 0, len(tenant.GetFinalizers()))

	namespaces, _ = namespacesClient.List(optList)
	// Only Tenant Namespace
	assert.Equal(t, 1, len(namespaces.Items))

}

func Test_TenantDeletion_WorksIfNamesapceWasDeletedBefore(t *testing.T) {

	const clientNamespace = "client1"
	const defaultTenantID = "tenant1"

	cf := fake.NewClientFactory(
		fake.NamespaceWithAnnotations(clientNamespace, map[string]string{
			steward.AnnotationTenantNamespacePrefix: prefix1,
			steward.AnnotationTenantRole:            defaultTenantRoleName,
		}),
		fake.ServiceAccount(defaultServiceAccountName, clientNamespace),
		fake.ClusterRole(defaultTenantRoleName),
	)

	//Start controller with initially existing K8s resources
	stopCh, controller := startController(t, cf)
	defer stopController(t, stopCh)

	tenantsClient := cf.StewardV1alpha1().Tenants(clientNamespace)
	namespacesClient := cf.CoreV1().Namespaces()

	t.Log("...CREATE TENANT")

	syncCount := controller.getSyncCount()
	_, err := tenantsClient.Create(fake.Tenant(defaultTenantID, "name", "dname", clientNamespace))
	assert.NilError(t, err)

	for controller.getSyncCount() <= syncCount {
		sleep(t, 100, "waiting for tenant controller sync")
	}
	tenants, _ := tenantsClient.List(optList)
	assert.Equal(t, 1, len(tenants.Items))

	t.Log("... DELETE NAMESPACE")

	tenant, err := tenantsClient.Get(defaultTenantID, optGet)
	assert.NilError(t, err)
	tenantNamespace := tenant.Status.TenantNamespaceName
	err = namespacesClient.Delete(tenantNamespace, optDelete)
	assert.NilError(t, err)
	namespaces, _ := namespacesClient.List(optList)
	// Client Namespace
	assert.Equal(t, 1, len(namespaces.Items))

	t.Log("... CHECK BEFORE DELETION")
	assert.Equal(t, 1, len(tenant.GetFinalizers()))

	t.Log("... DELETE TENANT")

	syncCount = controller.getSyncCount()

	// Fake client immediately deletes client
	// err = tenantsClient.Delete(tenant.GetName(), optDelete)
	// Set deletion timestamp instead
	now := metav1.Now()
	tenant.SetDeletionTimestamp(&now)
	_, err = tenantsClient.Update(tenant)

	assert.NilError(t, err)

	for controller.getSyncCount() <= syncCount {
		sleep(t, 100, "waiting for tenant controller sync")
	}

	t.Log("... CHECKS AFTER DELETE")

	tenant, _ = tenantsClient.Get(tenant.GetName(), optGet)
	// Consider tenant as deleted if deletion timestamp is set and finalizer list is empty
	assert.Assert(t, !tenant.GetDeletionTimestamp().IsZero())
	assert.Equal(t, 0, len(tenant.GetFinalizers()))

	namespaces, _ = namespacesClient.List(optList)
	// Only Tenant Namespace
	assert.Equal(t, 1, len(namespaces.Items))

}

func assertTenant(t *testing.T, cf *fake.ClientFactory, namespace string, expectedID string, expected expect) {
	tenantFetcher := k8s.NewTenantFetcher(cf)
	tenant, err := tenantFetcher.ByKey(tenantKey(namespace, expectedID))
	assert.NilError(t, err)
	tenantString := fmt.Sprintf(" ###### Tenant: %+v", tenant)

	assert.Equal(t, expectedID, tenant.GetName(), tenantString)
	if expected.name != "" {
		assert.Equal(t, expected.name, tenant.Spec.Name, tenantString)
	}
	if expected.displayName != "" {
		assert.Equal(t, expected.displayName, tenant.Spec.DisplayName, tenantString)
	}
	if expected.result != steward.TenantResult("") {
		assert.Equal(t, expected.result, tenant.Status.Result, tenantString)
	}
	if expected.message != "" {
		assert.Assert(t, strings.Contains(tenant.Status.Message, expected.message), fmt.Sprintf("'%s' not contained in '%s'", expected.message, tenant.Status.Message))
	}
	if expected.namespaceStartsWith != "" {
		if expected.namespaceStartsWith == "" {
			assert.Equal(t, "", tenant.Status.TenantNamespaceName, "TenantNamespaceName not empty")
		} else {
			assert.Assert(t, strings.HasPrefix(tenant.Status.TenantNamespaceName, expected.namespaceStartsWith), "Unexpected TenantNamespaceName: "+tenant.Status.TenantNamespaceName)
		}
	}

	if tenant.Status.TenantNamespaceName == "" {
		if expected.namespaceExists {
			assert.Assert(t, false, "tenant.Status.TenantNamespaceName is not set")
		}
	} else {
		assert.Assert(t, strings.HasPrefix(tenant.Status.TenantNamespaceName, expected.prefix+"-"+expectedID), "Unexpected TenantNamespaceName: "+tenant.Status.TenantNamespaceName)
		assertNamespace(t, cf, tenant.Status.TenantNamespaceName, expected.namespaceExists, expectedID)
	}
}

func assertNamespace(t *testing.T, cf *fake.ClientFactory, namespaceName string, namespaceExists bool, expectedID string) {
	printNamespaces(t, cf)
	assert.Assert(t, namespaceName != "", "namespaceName empty")
	namespace, err := cf.CoreV1().Namespaces().Get(namespaceName, optGet)
	if namespaceExists {
		assert.NilError(t, err)
		assert.Equal(t, namespaceName, namespace.GetObjectMeta().GetName())
		assert.Equal(t, expectedID, namespace.GetLabels()["id"])
	} else {
		assert.Assert(t, err != nil, "No error when getting namespace")
		assert.Assert(t, strings.Contains(err.Error(), "not found"), fmt.Sprintf("Not an 'namespace not found' error but '%s'", err.Error()))
	}
}

func printNamespaces(t *testing.T, cf *fake.ClientFactory) {
	namespaces, err := cf.CoreV1().Namespaces().List(optList)
	if err != nil {
		t.Logf("Error getting namespaces: %s", err.Error())
	} else {
		t.Log("Namespaces:")
		for _, n := range namespaces.Items {
			t.Logf("  {name:%s, owner:%v}", n.GetName(), n.GetOwnerReferences())
		}
	}
}

type expect struct {
	name                string
	displayName         string
	result              steward.TenantResult
	message             string
	prefix              string
	namespaceExists     bool
	namespaceStartsWith string
}

func fakeClusterRole() *v1beta1.ClusterRole {
	return fake.ClusterRole(defaultTenantRoleName)
}

func fakeServiceAccount() *v1.ServiceAccount {
	return fake.ServiceAccount(defaultServiceAccountName, ns1)
}

func startController(t *testing.T, cf *fake.ClientFactory) (chan struct{}, *Controller) {
	stopCh := make(chan struct{}, 0)
	metrics := NewMetrics()
	controller := NewController(cf, k8s.NewTenantFetcher(cf), metrics)
	cf.StewardInformerFactory().Start(stopCh)
	go start(t, controller, stopCh)
	cf.Sleep("Wait for controller")
	return stopCh, controller
}

func stopController(t *testing.T, stopCh chan struct{}) {
	t.Log("Trigger controller stop")
	stopCh <- struct{}{}
}

func start(t *testing.T, controller *Controller, stopCh chan struct{}) {
	if err := controller.Run(1, stopCh); err != nil {
		t.Logf("Error running controller %s", err.Error())
	}
}

func tenantKey(namespace string, tenantID string) string {
	return fmt.Sprintf("%s/%s", namespace, tenantID)
}

func sleep(t *testing.T, durationMillis int, message string) error {
	t.Log("wait for controller")
	duration, err := time.ParseDuration(fmt.Sprintf("%vms", durationMillis))
	if err != nil {
		return err
	}
	time.Sleep(duration)
	return nil
}
