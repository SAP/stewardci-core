package fake

import (
	"context"

	"github.com/SAP/stewardci-core/pkg/k8s/secrets/providers"
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
func (p *SecretProviderImpl) GetSecret(ctx context.Context, name string) (*v1.Secret, error) {
	for _, secret := range p.secrets {
		if secret.GetName() == name {
			if !secret.ObjectMeta.DeletionTimestamp.IsZero() {
				return nil, nil
			}
			secretCopy := secret.DeepCopy()
			providers.StripMetadata(secretCopy)
			return secretCopy, nil
		}
	}
	return nil, nil
}
