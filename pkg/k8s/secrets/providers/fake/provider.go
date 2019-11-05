package fake

import (
	"fmt"
	"strings"

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
func (p *SecretProviderImpl) GetSecret(name string) (*v1.Secret, error) {
	for _, secret := range p.secrets {
		if secret.GetName() == name {
			if !secret.ObjectMeta.DeletionTimestamp.IsZero() {
				return nil, fmt.Errorf("secret not found: %s/%s", p.namespace, name)
			}
			return providers.StripMetadata(secret), nil
		}
	}
	return nil, fmt.Errorf("secret not found: %s/%s", p.namespace, name)
}

// IsNotFound returns true if error returned from GetSecret is a not found error
func (p *SecretProviderImpl) IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "secret not found:")
}
