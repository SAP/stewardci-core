package k8s

import (
	"context"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func Test_provider_GetSecret_Existing(t *testing.T) {
	// SETUP
	ctx := context.Background()
	now := metav1.Now()
	var grace int64 = 1
	storedSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "foo",
			GenerateName:               "dummy",
			Namespace:                  "ns1",
			SelfLink:                   "dummy",
			UID:                        types.UID("dummy"),
			ResourceVersion:            "dummy",
			Generation:                 1,
			CreationTimestamp:          now,
			DeletionGracePeriodSeconds: &grace,
			OwnerReferences:            []metav1.OwnerReference{metav1.OwnerReference{}},
			Finalizers:                 []string{"dummy"},
			ClusterName:                "dummy",
			Labels: map[string]string{
				"lbar": "lbaz",
			},
			Annotations: map[string]string{
				"abar": "abaz",
			},
		},
		Type: v1.SecretTypeOpaque,
	}

	examinee := initProvider("ns1", storedSecret.DeepCopy())

	// EXERCISE
	resultSecret, resultErr := examinee.GetSecret(ctx, storedSecret.GetName())

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Assert(t, resultSecret != nil)

	expectedSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        storedSecret.GetName(),
			Labels:      storedSecret.GetLabels(),
			Annotations: storedSecret.GetAnnotations(),
		},
		Type: v1.SecretTypeOpaque,
	}

	assert.DeepEqual(t, *expectedSecret, *resultSecret)
}

func Test_provider_GetSecret_InDeletion(t *testing.T) {
	// SETUP
	ctx := context.Background()
	storedSecret := fake.SecretOpaque("foo", "ns1")
	now := metav1.Now()
	storedSecret.SetDeletionTimestamp(&now)

	examinee := initProvider("ns1", storedSecret)

	// EXERCISE
	resultSecret, resultErr := examinee.GetSecret(ctx, "foo")

	// VERIFY
	assert.Assert(t, resultErr == nil)
	assert.Assert(t, resultSecret == nil)
}

func Test_provider_GetSecret_NotExisting(t *testing.T) {
	// SETUP
	ctx := context.Background()
	examinee := initProvider("ns1" /* no secret exists */)

	// EXERCISE
	resultSecret, resultErr := examinee.GetSecret(ctx, "foo")

	// VERIFY
	assert.Assert(t, resultErr == nil)
	assert.Assert(t, resultSecret == nil)
}

func initProvider(namespace string, secrets ...*v1.Secret) secrets.SecretProvider {
	objects := make([]runtime.Object, len(secrets))
	for i, e := range secrets {
		objects[i] = e
	}
	cf := fake.NewClientFactory(objects...)
	secretsClient := cf.CoreV1().Secrets(namespace)
	return NewProvider(secretsClient, namespace)
}
