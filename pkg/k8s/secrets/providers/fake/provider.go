package fake

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

// SecretProviderImpl is an implementation of SecretProvider for testing purposes.
type SecretProviderImpl struct {
	namespace string
	secrets   []*v1.Secret
}

// NewProvider creates a fake secret provider for testing returning the secrets provided
func NewProvider(namespace string, secrets ...*v1.Secret) *SecretProviderImpl {
	return &SecretProviderImpl{
		namespace: namespace,
		secrets:   secrets,
	}
}

// GetSecret fulfills the SecretProvider interface.
func (p *SecretProviderImpl) GetSecret(name string) (*v1.Secret, error) {
	for _, secret := range p.secrets {
		if secret.GetName() == name {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("secret not found: %s/%s", p.namespace, name)
}
