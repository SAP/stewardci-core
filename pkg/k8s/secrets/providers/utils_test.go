package providers

import (
	"testing"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func Test_StripMetadata(t *testing.T) {
	// SETUP
	now := metav1.Now()
	var grace int64 = 1
	origSecret := &v1.Secret{
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
			OwnerReferences:            []metav1.OwnerReference{{}},
			Finalizers:                 []string{"dummy"},
			Labels: map[string]string{
				"lbar": "lbaz",
			},
			Annotations: map[string]string{
				"abar": "abaz",
			},
		},
		Type: v1.SecretTypeOpaque,
	}

	// EXERCISE
	resultSecret := origSecret.DeepCopy()
	StripMetadata(resultSecret)

	// VERIFY
	expectedSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        origSecret.GetName(),
			Labels:      origSecret.GetLabels(),
			Annotations: origSecret.GetAnnotations(),
		},
		Type: v1.SecretTypeOpaque,
	}

	assert.DeepEqual(t, *expectedSecret, *resultSecret)
}
