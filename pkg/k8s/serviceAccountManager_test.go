package k8s

import (
	"sync"
	"testing"
	"time"

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
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
	assert.Equal(t, "pipelineCloneSecretName1", acc.GetServiceAccount().Secrets[0].Name)
	assert.Equal(t, 2, len(acc.GetServiceAccount().ImagePullSecrets))
	assert.Equal(t, "imagePullSecret1", acc.GetServiceAccount().ImagePullSecrets[0].Name)
	assert.Equal(t, "imagePullSecret2", acc.GetServiceAccount().ImagePullSecrets[1].Name)
}

func Test_CreateServiceAccount_noPullSecrets(t *testing.T) {
	setupAccountManager()
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{})
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
	assert.Equal(t, "pipelineCloneSecretName1", acc.GetServiceAccount().Secrets[0].Name)
	assert.Equal(t, 0, len(acc.GetServiceAccount().ImagePullSecrets))
}

func Test_CreateServiceAccount_noCloneSecret(t *testing.T) {
	setupAccountManager()
	acc, err := accountManager.CreateServiceAccount(accountName, "", []string{"imagePullSecret1"})
	assert.NilError(t, err)
	assert.Equal(t, accountName, acc.GetServiceAccount().GetName())
	assert.Equal(t, 0, len(acc.GetServiceAccount().Secrets))
	assert.Equal(t, 1, len(acc.GetServiceAccount().ImagePullSecrets))
	assert.Equal(t, "imagePullSecret1", acc.GetServiceAccount().ImagePullSecrets[0].Name)

}

func Test_CreateServiceAccount_failsWhenAlreadyExists(t *testing.T) {
	setupAccountManager(fakeServiceAccount())
	_, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecretName1"})
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

func Test_SetDoAutomountServiceAccountToken_works(t *testing.T) {
	//SETUP
	fakeServiceAccount := fakeServiceAccount()
	setupAccountManager(fakeServiceAccount)
	acc, err := accountManager.GetServiceAccount(accountName)
	assert.NilError(t, err)

	//EXERCISE
	acc.SetDoAutomountServiceAccountToken(false)
	assert.NilError(t, acc.Update())

	//VERIFY
	actual, err := accountManager.GetServiceAccount(accountName)
	assert.NilError(t, err)
	assert.Check(t, *actual.cache.AutomountServiceAccountToken == false)
}

func Test_GetServiceAccountSecretName_works(t *testing.T) {
	//SETUP
	secretName := "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}
	setupAccountManager(secret)
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	assert.NilError(t, err)

	acc.AttachSecrets("a-secret", secretName, "z-secret")

	// EXERCISE
	resultName := acc.GetServiceAccountSecretName()
	// VERIFY
	assert.Equal(t, secretName, resultName)
}

func Test_GetServiceAccountSecretNameRepeat_delayedRef_works(t *testing.T) {
	//SETUP
	secretName := "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}
	setupAccountManager(secret)
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	assert.NilError(t, err)
	err = acc.Update()
	assert.NilError(t, err)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func(t *testing.T, acc *ServiceAccountWrap) {
		defer waitGroup.Done()
		// EXERCISE
		resultName := acc.GetServiceAccountSecretNameRepeat()
		// VERIFY
		assert.Equal(t, "ns1-token-foo", resultName)
	}(t, acc)
	duration, _ := time.ParseDuration("500ms")
	time.Sleep(duration)
	localAccountManager := NewServiceAccountManager(factory, ns1)
	localAccount, err := localAccountManager.GetServiceAccount(accountName)
	assert.NilError(t, err)
	localAccount.AttachSecrets("a-secret", secretName, "z-secret")
	err = localAccount.Update()
	assert.NilError(t, err)
	waitGroup.Wait()
}

func Test_GetServiceAccountSecretNameRepeat_delayedSecret_works(t *testing.T) {
	//SETUP
	secretName := "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}
	setupAccountManager()
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	acc.AttachSecrets("a-secret", secretName, "z-secret")
	assert.NilError(t, err)
	err = acc.Update()
	assert.NilError(t, err)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func(t *testing.T, acc *ServiceAccountWrap) {
		defer waitGroup.Done()
		// EXERCISE
		resultName := acc.GetServiceAccountSecretNameRepeat()
		// VERIFY
		assert.Equal(t, "ns1-token-foo", resultName)
	}(t, acc)
	duration, _ := time.ParseDuration("500ms")
	time.Sleep(duration)
	secretsInterface := factory.CoreV1().Secrets(ns1)
	_, err = secretsInterface.Create(secret)
	assert.NilError(t, err)
	waitGroup.Wait()
}

func Test_GetServiceAccountSecretName_wrongType(t *testing.T) {
	//SETUP
	secretName := "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeOpaque,
	}
	setupAccountManager(secret)
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	assert.NilError(t, err)

	acc.AttachSecrets("a-secret", secretName, "z-secret")

	// EXERCISE
	resultName := acc.GetServiceAccountSecretName()
	// VERIFY
	assert.Equal(t, "", resultName)
}

func Test_GetServiceAccountSecretName_ref_missing(t *testing.T) {
	//SETUP
	secretName := "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}
	setupAccountManager(secret)
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	assert.NilError(t, err)

	acc.AttachSecrets("a-secret", "z-secret")

	// EXERCISE
	resultName := acc.GetServiceAccountSecretName()
	// VERIFY
	assert.Equal(t, "", resultName)
}

func Test_GetServiceAccountSecretName_secret_missing(t *testing.T) {
	//SETUP
	secretName := "ns1-token-foo"
	setupAccountManager()
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	assert.NilError(t, err)

	acc.AttachSecrets("a-secret", secretName, "z-secret")

	// EXERCISE
	resultName := acc.GetServiceAccountSecretName()
	// VERIFY
	assert.Equal(t, "", resultName)
}
