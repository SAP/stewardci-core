package providers

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StripMetadata strips the metadata from a secret
func StripMetadata(secret *v1.Secret) {
	secret.ObjectMeta = metav1.ObjectMeta{
		Name:        secret.GetName(),
		Labels:      secret.GetLabels(),
		Annotations: secret.GetAnnotations(),
	}
}
