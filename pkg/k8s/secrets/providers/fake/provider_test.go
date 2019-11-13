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
	storedSecret := fake.SecretWithMetadata("foo", "ns1", v1.SecretTypeOpaque)
	labels := map[string]string{"lbar": "lbaz"}
	storedSecret.SetLabels(labels)
	annotations := map[string]string{"abar": "abaz"}
	storedSecret.SetAnnotations(annotations)
	provider := initProvider("ns1", storedSecret)
	providedSecret, err := provider.GetSecret("foo")
	assert.NilError(t, err)
	assert.Equal(t, "foo", providedSecret.GetName())
	assert.Equal(t, "", providedSecret.GetGenerateName())
	assert.Equal(t, "", providedSecret.GetNamespace())
	assert.Equal(t, "", providedSecret.GetSelfLink())
	assert.Equal(t, types.UID(""), providedSecret.GetUID())
	assert.Equal(t, "", providedSecret.GetResourceVersion())
	assert.Equal(t, int64(0), providedSecret.GetGeneration())

	assert.DeepEqual(t, annotations, providedSecret.GetAnnotations())
	assert.DeepEqual(t, labels, providedSecret.GetLabels())
	creationTime := providedSecret.GetCreationTimestamp()
	assert.Assert(t, (&creationTime).IsZero())
	assert.Assert(t, providedSecret.GetDeletionTimestamp().IsZero())
}

func Test_provider_GetSecret_InDeletion(t *testing.T) {
	deletedSecret := fake.Secret("foo", "ns1")
	now := metav1.Now()
	deletedSecret.SetDeletionTimestamp(&now)
	provider := initProvider("ns1", deletedSecret)
	secret, err := provider.GetSecret("foo")
	assert.Assert(t, err == nil)
	assert.Assert(t, secret == nil)

}

func Test_provider_GetSecret_NotExisting(t *testing.T) {
	provider := initProvider("ns1", fake.Secret("foo", "ns1"))
	secret, err := provider.GetSecret("bar")
	assert.Assert(t, err == nil)
	assert.Assert(t, secret == nil)
}

func initProvider(namespace string, secret *v1.Secret) secrets.SecretProvider {
	return NewProvider(namespace, secret)
}
