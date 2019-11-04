package fake

import (
	v1 "k8s.io/api/core/v1"
)

// Secret creates a fake secret with defined name
func Secret(name string, namespace string) *v1.Secret {
	return &v1.Secret{ObjectMeta: ObjectMeta(name, namespace)}
}

// SecretWithType creates a fake secret with defined name
func SecretWithType(name string, namespace string, secretType v1.SecretType) *v1.Secret {
	return &v1.Secret{ObjectMeta: ObjectMeta(name, namespace), Type: secretType}
}
