package secrets

import (
	v1 "k8s.io/api/core/v1"
)

// SecretProvider provides secrets
type SecretProvider interface {
	// GetSecret returns a secret by its name
	// returns nil,nil if secret is not found
	GetSecret(name string) (*v1.Secret, error)
}
