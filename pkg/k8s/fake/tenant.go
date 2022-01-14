package fake

import (
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tenant creates a new fake tenant object.
func Tenant(name, namespace string) *stewardv1alpha1.Tenant {
	return &stewardv1alpha1.Tenant{
		TypeMeta: metav1.TypeMeta{
			APIVersion: stewardv1alpha1.SchemeGroupVersion.String(),
			Kind:       "Tenant",
		},
		ObjectMeta: ObjectMeta(name, namespace),
	}
}
