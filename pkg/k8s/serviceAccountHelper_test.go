package k8s

import (
	"context"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_GetServiceAccountSecretName_works(t *testing.T) {
	t.Parallel()
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
	err = acc.Update()
	assert.NilError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	ctx = WithClientFactory(ctx, factory)
	ctx = EnsureServiceAccountTokenSecretRetriever(ctx)
	examinee := GetServiceAccountTokenSecretRetriever(ctx)
	// EXERCISE
	result, err := examinee.ForName(ctx, accountName, ns1)
	// VERIFY
	assert.NilError(t, err)
	assert.DeepEqual(t, secret, result)
}

func Test_GetServiceAccountSecretNameRepeat_delayedRef_works(t *testing.T) {
	t.Parallel()
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
		ctx := WithClientFactory(context.TODO(), factory)
		ctx = EnsureServiceAccountTokenSecretRetriever(ctx)
		examinee := GetServiceAccountTokenSecretRetriever(ctx)
		// EXERCISE
		result, err := examinee.ForName(ctx, accountName, ns1)

		// VERIFY
		assert.NilError(t, err)
		assert.Equal(t, "ns1-token-foo", result.GetName())
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
	t.Parallel()
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
		ctx := WithClientFactory(context.TODO(), factory)
		ctx = EnsureServiceAccountTokenSecretRetriever(ctx)
		examinee := GetServiceAccountTokenSecretRetriever(ctx)
		// EXERCISE
		result, err := examinee.ForName(ctx, accountName, ns1)

		// VERIFY
		assert.NilError(t, err)
		assert.Equal(t, "ns1-token-foo", result.GetName())
	}(t, acc)
	duration, _ := time.ParseDuration("500ms")
	time.Sleep(duration)
	secretsInterface := factory.CoreV1().Secrets(ns1)
	_, err = secretsInterface.Create(secret)
	assert.NilError(t, err)
	waitGroup.Wait()
}

func Test_GetServiceAccountSecretName_wrongType(t *testing.T) {
	t.Parallel()
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
	err = acc.Update()
	assert.NilError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	ctx = WithClientFactory(ctx, factory)
	ctx = EnsureServiceAccountTokenSecretRetriever(ctx)
	examinee := GetServiceAccountTokenSecretRetriever(ctx)
	// EXERCISE
	result, err := examinee.ForName(ctx, accountName, ns1)

	// VERIFY
	assert.Equal(t, err.Error(), "context deadline exceeded")
	assert.Assert(t, nil == result)
}

func Test_GetServiceAccountSecretName_ref_missing(t *testing.T) {
	t.Parallel()
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
	err = acc.Update()
	assert.NilError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	ctx = WithClientFactory(ctx, factory)
	ctx = EnsureServiceAccountTokenSecretRetriever(ctx)
	examinee := GetServiceAccountTokenSecretRetriever(ctx)
	// EXERCISE
	result, err := examinee.ForName(ctx, accountName, ns1)

	// VERIFY
	assert.Equal(t, err.Error(), "context deadline exceeded")
	assert.Assert(t, nil == result)
}

func Test_GetServiceAccountSecretName_secret_missing(t *testing.T) {
	t.Parallel()
	//SETUP
	secretName := "ns1-token-foo"
	setupAccountManager()
	acc, err := accountManager.CreateServiceAccount(accountName, "pipelineCloneSecretName1", []string{"imagePullSecret1", "imagePullSecret2"})
	assert.NilError(t, err)

	acc.AttachSecrets("a-secret", secretName, "z-secret")
	err = acc.Update()
	assert.NilError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	ctx = WithClientFactory(ctx, factory)
	ctx = EnsureServiceAccountTokenSecretRetriever(ctx)
	examinee := GetServiceAccountTokenSecretRetriever(ctx)
	// EXERCISE
	result, err := examinee.ForName(ctx, accountName, ns1)

	// VERIFY
	assert.Equal(t, err.Error(), "context deadline exceeded")
	assert.Assert(t, nil == result)
}
