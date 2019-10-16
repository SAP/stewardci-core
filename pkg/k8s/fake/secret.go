package fake

import (
	v1 "k8s.io/api/core/v1"
)

// SecretKey is a fake key
const SecretKey string = "Key"

// SecretValue is a fake value
const SecretValue string = "Value"

// Secret creates a fake secret with defined name
func Secret(name string, namespace string) *v1.Secret {
	data := make(map[string]string)
	data[SecretKey] = SecretValue
	return &v1.Secret{ObjectMeta: ObjectMeta(name, namespace), StringData: data}
}
