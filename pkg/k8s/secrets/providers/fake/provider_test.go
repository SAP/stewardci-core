package fake

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func Test_provider_GetSecret_Existing(t *testing.T) {
	// SETUP
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
	resultSecret, resultErr := examinee.GetSecret(storedSecret.GetName())

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
	deletedSecret := fake.SecretOpaque("foo", "ns1")
	now := metav1.Now()
	deletedSecret.SetDeletionTimestamp(&now)
	provider := initProvider("ns1", deletedSecret)
	secret, err := provider.GetSecret("foo")
	assert.Assert(t, err == nil)
	assert.Assert(t, secret == nil)

}

func Test_provider_GetSecret_NotExisting(t *testing.T) {
	provider := initProvider("ns1", fake.SecretOpaque("foo", "ns1"))
	secret, err := provider.GetSecret("bar")
	assert.Assert(t, err == nil)
	assert.Assert(t, secret == nil)
}

func initProvider(namespace string, secret *v1.Secret) secrets.SecretProvider {
	return NewProvider(namespace, secret)
}
