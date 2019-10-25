package k8s

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

var accountManager ServiceAccountManager
var factory ClientFactory

const (
	accountName = "dummyAccount"
	roleName    = RoleName("dummyRole")
)

func setupAccountManager(objects ...runtime.Object) {
	factory = fake.NewClientFactory(objects...)
	accountManager = NewServiceAccountManager(factory, ns1)
}

func fakeServiceAccount() *v1.ServiceAccount {
	return &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: accountName, Namespace: ns1}}
}

func Test_CreateServiceAccount_works(t *testing.T) {
	setupAccountManager()
	acc, err := accountManager.CreateServiceAccount(accountName, "scmCloneSecretName", []string{"pullSecretName"})
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
}

func Test_CreateServiceAccount_failsWhenAlreadyExists(t *testing.T) {
	setupAccountManager(fakeServiceAccount())
	_, err := accountManager.CreateServiceAccount(accountName, "scmCloneSecretName", []string{"pullSecretName"})
	assert.Equal(t, `serviceaccounts "dummyAccount" already exists`, err.Error())
}

func Test_FetchServiceAccount_works(t *testing.T) {
	setupAccountManager(fakeServiceAccount())
	acc, err := accountManager.GetServiceAccount(accountName)
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
}

func Test_FetchServiceAccount_failsIfNotExisting(t *testing.T) {
	setupAccountManager()
	_, err := accountManager.GetServiceAccount(accountName)
	assert.Equal(t, `serviceaccounts "dummyAccount" not found`, err.Error())
}

func Test_CreateRoleSameNamespace_works(t *testing.T) {
	setupAccountManager(fakeServiceAccount(), fake.ClusterRole(string(roleName)))
	acc, _ := accountManager.GetServiceAccount(accountName)
	_, err := acc.AddRoleBinding(roleName, ns1)
	assert.NilError(t, err)
}

func Test_CreateRoleOtherNamespace_works(t *testing.T) {
	setupAccountManager(fakeServiceAccount(), fake.ClusterRole(string(roleName)))
	acc, _ := accountManager.GetServiceAccount(accountName)
	_, err := acc.AddRoleBinding(roleName, ns1)
	assert.NilError(t, err)
}
