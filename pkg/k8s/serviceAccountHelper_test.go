package k8s

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_serviceAccountHelper_GetServiceAccountSecretName_works(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()

	const secretName = "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}

	cf := fake.NewClientFactory(
		secret,
	)
	accountManager := NewServiceAccountManager(cf, ns1)
	account, err := accountManager.CreateServiceAccount(
		ctx,
		accountName,
		"pipelineCloneSecretName1",
		[]string{
			"imagePullSecret1",
			"imagePullSecret2",
		},
	)
	assert.NilError(t, err)

	account.AttachSecrets(
		"a-secret",
		secretName,
		"z-secret",
	)
	examinee := account.GetHelper()

	// EXERCISE
	result, resultErr := examinee.GetServiceAccountSecretName(ctx)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, secretName, result)
}

func Test_serviceAccountHelper_GetServiceAccountSecretNameRepeat_delayedRef_works(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()

	const secretName = "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}

	cf := fake.NewClientFactory(
		secret,
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	account, err := accountManager.CreateServiceAccount(
		ctx,
		accountName,
		"pipelineCloneSecretName1",
		[]string{
			"imagePullSecret1",
			"imagePullSecret2",
		},
	)
	assert.NilError(t, err)
	err = account.Update(ctx)
	assert.NilError(t, err)

	examinee := account.GetHelper()

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	// attach secret concurrently with delay
	go func() {
		defer waitGroup.Done()

		time.Sleep(100 * time.Millisecond)

		localAccountManager := NewServiceAccountManager(cf, ns1)
		localAccount, err := localAccountManager.GetServiceAccount(ctx, accountName)
		assert.NilError(t, err)

		localAccount.AttachSecrets(
			"a-secret",
			secretName,
			"z-secret",
		)
		err = localAccount.Update(ctx)
		assert.NilError(t, err)
	}()

	// EXERCISE
	result, resultErr := examinee.GetServiceAccountSecretNameRepeat(ctx)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, secretName, result)

	waitGroup.Wait()
}

func Test_serviceAccountHelper_GetServiceAccountSecretNameRepeat_delayedSecret_works(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()

	const secretName = "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}

	cf := fake.NewClientFactory()
	accountManager := NewServiceAccountManager(cf, ns1)

	account, err := accountManager.CreateServiceAccount(
		ctx,
		accountName,
		"pipelineCloneSecretName1",
		[]string{
			"imagePullSecret1",
			"imagePullSecret2",
		},
	)
	account.AttachSecrets(
		"a-secret",
		secretName,
		"z-secret",
	)
	assert.NilError(t, err)
	err = account.Update(ctx)
	assert.NilError(t, err)

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	examinee := account.GetHelper()

	// create secret concurrently with delay
	go func() {
		defer waitGroup.Done()

		time.Sleep(100 * time.Millisecond)

		secretsInterface := cf.CoreV1().Secrets(ns1)
		_, err = secretsInterface.Create(ctx, secret, metav1.CreateOptions{})
		assert.NilError(t, err)
	}()

	// EXERCISE
	result, resultErr := examinee.GetServiceAccountSecretNameRepeat(ctx)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, secretName, result)

	waitGroup.Wait()
}

func Test_serviceAccountHelper_GetServiceAccountSecretName_wrongType(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()

	const secretName = "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeOpaque,
	}

	cf := fake.NewClientFactory(
		secret,
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	account, err := accountManager.CreateServiceAccount(
		ctx,
		accountName,
		"pipelineCloneSecretName1",
		[]string{
			"imagePullSecret1",
			"imagePullSecret2",
		},
	)
	assert.NilError(t, err)

	account.AttachSecrets(
		"a-secret",
		secretName,
		"z-secret",
	)
	examinee := account.GetHelper()

	// EXERCISE
	result, resultErr := examinee.GetServiceAccountSecretName(ctx)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, "", result)
}

func Test_serviceAccountHelper_GetServiceAccountSecretName_refMissing(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()

	const secretName = "ns1-token-foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns1,
		},
		Type: v1.SecretTypeServiceAccountToken,
	}

	cf := fake.NewClientFactory(
		secret,
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	account, err := accountManager.CreateServiceAccount(
		ctx,
		accountName,
		"pipelineCloneSecretName1",
		[]string{
			"imagePullSecret1",
			"imagePullSecret2",
		},
	)
	assert.NilError(t, err)

	account.AttachSecrets(
		"a-secret",
		"z-secret",
	)
	examinee := account.GetHelper()

	// EXERCISE
	result, resultErr := examinee.GetServiceAccountSecretName(ctx)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, "", result)
}

func Test_serviceAccountHelper_GetServiceAccountSecretName_secretMissing(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()

	const secretName = "ns1-token-foo"

	cf := fake.NewClientFactory(
	// no secret here
	)
	accountManager := NewServiceAccountManager(cf, ns1)

	account, err := accountManager.CreateServiceAccount(
		ctx,
		accountName,
		"pipelineCloneSecretName1",
		[]string{
			"imagePullSecret1",
			"imagePullSecret2",
		},
	)
	assert.NilError(t, err)

	account.AttachSecrets(
		"a-secret",
		secretName,
		"z-secret",
	)
	examinee := account.GetHelper()

	// EXERCISE
	result, resultErr := examinee.GetServiceAccountSecretName(ctx)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Equal(t, "", result)
}
