package k8s

import (
	"context"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	accountName = "dummyAccount"
	roleName    = RoleName("dummyRole")
)

func fakeServiceAccount() *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: accountName, Namespace: ns1,
		},
	}
}

func Test_serviceAccountManager_CreateServiceAccount_works(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory()
	examinee := NewServiceAccountManager(cf, ns1)

	// EXERCISE
	result, resultErr := examinee.CreateServiceAccount(ctx, accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, accountName, result.GetServiceAccount().GetName())
	assert.Equal(t, "pipelineCloneSecretName1", result.GetServiceAccount().Secrets[0].Name)
	assert.Equal(t, 2, len(result.GetServiceAccount().ImagePullSecrets))
	assert.Equal(t, "imagePullSecret1", result.GetServiceAccount().ImagePullSecrets[0].Name)
	assert.Equal(t, "imagePullSecret2", result.GetServiceAccount().ImagePullSecrets[1].Name)
}

func Test_serviceAccountManager_CreateServiceAccount_noPullSecrets(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory()
	accountManager := NewServiceAccountManager(cf, ns1)

	// EXERCISE
	acc, err := accountManager.CreateServiceAccount(ctx, accountName, "pipelineCloneSecretName1", []string{})

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
	assert.Equal(t, "pipelineCloneSecretName1", acc.GetServiceAccount().Secrets[0].Name)
	assert.Equal(t, 0, len(acc.GetServiceAccount().ImagePullSecrets))
}

func Test_serviceAccountManager_CreateServiceAccount_noCloneSecret(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory()
	examinee := NewServiceAccountManager(cf, ns1)

	// EXERCISE
	acc, err := examinee.CreateServiceAccount(ctx, accountName, "", []string{"imagePullSecret1"})

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
	assert.Equal(t, 0, len(acc.GetServiceAccount().Secrets))
	assert.Equal(t, 1, len(acc.GetServiceAccount().ImagePullSecrets))
	assert.Equal(t, "imagePullSecret1", acc.GetServiceAccount().ImagePullSecrets[0].Name)

}

func Test_serviceAccountManager_CreateServiceAccount_failsWhenAlreadyExists(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		fakeServiceAccount(),
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	// EXERCISE
	_, err := accountManager.CreateServiceAccount(ctx, accountName, "pipelineCloneSecretName1", []string{"imagePullSecretName1"})

	// VERIFY
	assert.Equal(t, `serviceaccounts "dummyAccount" already exists`, err.Error())
}

func Test_serviceAccountManager_FetchServiceAccount_Exists(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		fakeServiceAccount(),
	)
	examinee := NewServiceAccountManager(cf, ns1)

	// EXERCISE
	acc, err := examinee.GetServiceAccount(ctx, accountName)

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
}

func Test_serviceAccountManager_FetchServiceAccount_NotExisting(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory()
	examinee := NewServiceAccountManager(cf, ns1)

	// EXERCISE
	_, err := examinee.GetServiceAccount(ctx, accountName)

	// VERIFY
	assert.Equal(t, `serviceaccounts "dummyAccount" not found`, err.Error())
}

func Test_ServiceAccountWrap_AddRoleBinding_SameNamespace(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		fakeServiceAccount(),
		fake.ClusterRole(string(roleName)),
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	examinee, err := accountManager.GetServiceAccount(ctx, accountName)
	assert.NilError(t, err)

	// EXERCISE
	_, resultErr := examinee.AddRoleBinding(ctx, roleName, ns1)

	// VERIFY
	assert.NilError(t, resultErr)
	// TODO verify result
}

func Test_ServiceAccountWrap_AddRoleBinding_CreateRoleOtherNamespace_works(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		fakeServiceAccount(),
		fake.ClusterRole(string(roleName)),
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	examinee, err := accountManager.GetServiceAccount(ctx, accountName)
	assert.NilError(t, err)

	// EXERCISE
	_, resultErr := examinee.AddRoleBinding(ctx, roleName, ns1)

	// VERIFY
	assert.NilError(t, resultErr)
	// TODO verify result
}

func Test_ServiceAccountWrap_SetDoAutomountServiceAccountToken(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
		fakeServiceAccount(),
		fake.ClusterRole(string(roleName)),
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	examinee, err := accountManager.GetServiceAccount(ctx, accountName)
	assert.NilError(t, err)

	// EXERCISE
	examinee.SetDoAutomountServiceAccountToken(false)

	// VERIFY
	assert.NilError(t, examinee.Update(ctx))
	actual, err := accountManager.GetServiceAccount(ctx, accountName)
	assert.NilError(t, err)
	assert.Assert(t, *actual.cache.AutomountServiceAccountToken == false)
}
