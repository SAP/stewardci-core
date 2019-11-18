package providers

import (
	"testing"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_StripMetadata(t *testing.T) {
	now := metav1.Now()
	var grace int64 = 1
	secret := &v1.Secret{
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
		},
		Type: v1.SecretTypeOpaque,
	}
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
