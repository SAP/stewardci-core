package providers

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func Test_StripMetadata(t *testing.T) {
	secret := fake.SecretWithMetadata("foo", "ns1", v1.SecretTypeOpaque)
	labels := map[string]string{"lbar": "lbaz"}
	secret.SetLabels(labels)
	annotations := map[string]string{"abar": "abaz"}
	secret.SetAnnotations(annotations)

	stripedSecret := StripMetadata(secret)

	assert.Equal(t, "foo", stripedSecret.GetName())
	assert.Equal(t, "", stripedSecret.GetGenerateName())
	assert.Equal(t, "", stripedSecret.GetNamespace())
	assert.Equal(t, "", stripedSecret.GetSelfLink())
	assert.Equal(t, types.UID(""), stripedSecret.GetUID())
	assert.Equal(t, "", stripedSecret.GetResourceVersion())
	assert.Equal(t, int64(0), stripedSecret.GetGeneration())

	assert.DeepEqual(t, annotations, stripedSecret.GetAnnotations())
	assert.DeepEqual(t, labels, stripedSecret.GetLabels())
	creationTime := stripedSecret.GetCreationTimestamp()
	assert.Assert(t, (&creationTime).IsZero())
	assert.Assert(t, stripedSecret.GetDeletionTimestamp().IsZero())
}
