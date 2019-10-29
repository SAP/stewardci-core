package secrets

import (
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
)

// SecretHelper copies secrets
type SecretHelper interface {
	CopySecrets(secretNames []string, filter secrets.SecretFilterType, transformers ...secrets.SecretTransformerType) ([]string, error)
	CreateSecret(secret *v1.Secret) *v1.Secret
}

type secretHelper struct {
	provider  secrets.SecretProvider
	namespace string
	client    corev1.SecretInterface
}

// NewSecretHelper creates a secret helper
func NewSecretHelper(provider secrets.SecretProvider, namespace string, client corev1.SecretInterface) SecretHelper {
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
func (h *secretHelper) CopySecrets(secretNames []string, filter secrets.SecretFilterType, transformers ...secrets.SecretTransformerType) ([]string, error) {
	var storedSecretNames []string
	for _, secretName := range secretNames {
		secret, err := h.provider.GetSecret(secretName)
		if err != nil {
			return storedSecretNames, err
		}
		if filter != nil && !filter(secret) {
			continue
		}
		for _, transformer := range transformers {
			secret = transformer(secret)
		}
		storedSecret := h.CreateSecret(secret)
		storedSecretNames = append(storedSecretNames, storedSecret.GetName())
	}
	return storedSecretNames, nil
}

// CreateSecret stores the given secret
func (h *secretHelper) CreateSecret(secret *v1.Secret) *v1.Secret {
	newSecret := &v1.Secret{Data: secret.Data, StringData: secret.StringData, Type: secret.Type}
	name := secret.GetName()
	newSecret.SetName(name)
	newSecret.SetNamespace(h.namespace)
	newSecret.SetLabels(secret.GetLabels())
	newSecret.SetAnnotations(secret.GetAnnotations())
	secret, err := h.client.Create(newSecret)
	if err != nil {
		log.Printf("Cannot create secret '%s' in namespace '%s': %s", name, h.namespace, err)
	}
	log.Printf("Copy secret: %s", name)
	return secret
}
