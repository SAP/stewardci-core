package providers

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StripMetadata strips the metadata from a secret
func StripMetadata(secret *v1.Secret) *v1.Secret {
	newSecret := secret.DeepCopy()
	newSecret.ObjectMeta = metav1.ObjectMeta{
		Name:        secret.GetName(),
		Labels:      secret.GetLabels(),
		Annotations: secret.GetAnnotations(),
	}
	return newSecret
}
