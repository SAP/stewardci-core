package k8s

import (
	"fmt"
	"log"
	"strings"

	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// SecretHelper copies secrets
type SecretHelper interface {
	CopySecrets(secretNames []string, filter SecretFilterType, transformers ...func(*v1.Secret) *v1.Secret) ([]string, error)
	CreateSecret(secret *v1.Secret) *v1.Secret
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

// SecretFilterType is a type for filter function
// true  -> keep item
// false -> skip item
// filter function nil keeps all items
type SecretFilterType = func(*v1.Secret) bool

// SecretTransformerType is a type for secret transformers
type SecretTransformerType = func(*v1.Secret) *v1.Secret

// CopySecrets copies a let of secrets with defined names
// filter can be defined to copy only dedicated secrets
// transformers can be defined to transform the secrets before they are stored
// returns a list of the secret names (after transformation) which were stored
func (h *secretHelper) CopySecrets(secretNames []string, filter SecretFilterType, transformers ...func(*v1.Secret) *v1.Secret) ([]string, error) {
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

// DockerOnly filter to filter only docker secrets
var DockerOnly SecretFilterType = func(secret *v1.Secret) bool {
	return secret.Type == v1.SecretTypeDockerConfigJson || secret.Type == v1.SecretTypeDockercfg
}

// AppendNameSuffixFunc returns a mapping function from secret to secret
// in the result the secret has a new name with suffix 'suffix'
func AppendNameSuffixFunc(suffix string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		secret.SetName(fmt.Sprintf("%s-%s", secret.GetName(), suffix))
		return secret
	}
}

// SetAnnotationFunc returns a mapping function from secret to secret
// in the result secret the annotation with key 'key' is set to the value 'value'.
func SetAnnotationFunc(key string, value string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		annotations := secret.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[key] = value
		secret.SetAnnotations(annotations)
		return secret
	}
}

// StripAnnotationsFunc returns a mapping function from secret to secret
// in the result secret all annotations with prefix 'keyPrefix' are removed.
func StripAnnotationsFunc(keyPrefix string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		annotations := secret.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		for key := range annotations {
			if strings.HasPrefix(key, keyPrefix) {
				delete(annotations, key)
			}
		}
		secret.SetAnnotations(annotations)
		return secret
	}
}

// SetLabelFunc returns a mapping function from secret to secret
// in the result secret the label with key 'key' is set to the value 'value'.
func SetLabelFunc(key string, value string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		labels := secret.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[key] = value
		secret.SetLabels(labels)
		return secret
	}
}

// StripLabelsFunc returns a mapping function from secret to secret
// in the result secret all labels with prefix 'keyPrefix' are removed.
func StripLabelsFunc(keyPrefix string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		labels := secret.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		for key := range labels {
			if strings.HasPrefix(key, keyPrefix) {
				delete(labels, key)
			}
		}
		secret.SetLabels(labels)
		return secret
	}
}
