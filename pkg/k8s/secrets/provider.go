package secrets

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

// SecretProvider provides secrets
type SecretProvider interface {
	// GetSecret returns a secret by its name
	// returns nil,nil if secret is not found
	GetSecret(ctx context.Context, name string) (*v1.Secret, error)
}
