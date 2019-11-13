package secrets

import (
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// SecretHelper copies secrets
type SecretHelper interface {
	CopySecrets(secretNames []string, filter SecretFilterType, transformers ...SecretTransformerType) ([]string, error)
	CreateSecret(secret *v1.Secret) (*v1.Secret, error)
	IsNotFound(err error) bool
}

type secretHelper struct {
	provider  SecretProvider
	namespace string
	client    corev1.SecretInterface
}

// NewSecretHelper creates a secret helper
func NewSecretHelper(provider SecretProvider, namespace string, client corev1.SecretInterface) SecretHelper {
	return &secretHelper{
		provider:  provider,
		namespace: namespace,
		client:    client,
	}
}

// CopySecrets copies a set of secrets with defined names
// filter can be defined to copy only dedicated secrets
// transformers can be defined to transform the secrets before they are stored
// returns a list of the secret names (after transformation) which were stored
// In case of an error the copying is stopped. The result list contains the secrets already copied
// before the error occured. There is no rollback done by this function.
func (h *secretHelper) CopySecrets(secretNames []string, filter SecretFilterType, transformers ...SecretTransformerType) ([]string, error) {
	var storedSecretNames []string
	for _, secretName := range secretNames {
		secret, err := h.provider.GetSecret(secretName)
		if err != nil {
			return storedSecretNames, err
		}
		if secret == nil {
			return storedSecretNames, NewNotFoundError(secretName)
		}
		if filter != nil && !filter(secret) {
			continue
		}
		for _, transformer := range transformers {
			secret = transformer(secret)
		}
		storedSecret, err := h.CreateSecret(secret)
		if err != nil {
			return storedSecretNames, err
		}
		storedSecretNames = append(storedSecretNames, storedSecret.GetName())
	}
	return storedSecretNames, nil
}

type notFoundError struct {
	name string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("secret not found: '%s'", e.name)
}

// NewNotFoundError returns a not found error
func NewNotFoundError(name string) error {
	return &notFoundError{name: name}
}

// IsNotFound returns true if the error returned by CopySecrets is a not found error
func (h *secretHelper) IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	default:
		cause := errors.Cause(err)
		if err == cause {
			return false
		}
		return h.IsNotFound(cause)
	case *notFoundError:
		return true
	}
}

// CreateSecret creates the given secret in the storage the underlying client is connected to.
func (h *secretHelper) CreateSecret(secret *v1.Secret) (*v1.Secret, error) {
	newSecret := &v1.Secret{Data: secret.Data, StringData: secret.StringData, Type: secret.Type}
	name := secret.GetName()
	newSecret.SetName(name)
	newSecret.SetNamespace(h.namespace)
	newSecret.SetLabels(secret.GetLabels())
	newSecret.SetAnnotations(secret.GetAnnotations())
	return h.client.Create(newSecret)
}
