package secrets

import (
	v1 "k8s.io/api/core/v1"
)

// SecretProvider provides secrets
type SecretProvider interface {
	GetSecret(name string) (*v1.Secret, error)
}
