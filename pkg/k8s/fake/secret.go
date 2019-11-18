package fake

import (
	v1 "k8s.io/api/core/v1"
)

// SecretOpaque creates a fake secret with defined name and type Opaque
func SecretOpaque(name string, namespace string) *v1.Secret {
	return &v1.Secret{ObjectMeta: ObjectMeta(name, namespace), Type: v1.SecretTypeOpaque}
}

// SecretWithType creates a fake secret with defined name
func SecretWithType(name string, namespace string, secretType v1.SecretType) *v1.Secret {
	return &v1.Secret{ObjectMeta: ObjectMeta(name, namespace), Type: secretType}
}

//SecretWithMetadata creates a fake secret with metadata
func SecretWithMetadata(name, namespace string, secretType v1.SecretType) *v1.Secret {
	return &v1.Secret{ObjectMeta: ObjectMetaFull(name, namespace), Type: secretType}
}
