package k8s

import (
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"log"
)

type provider struct {
	namespace     string
	secretsClient corev1.SecretInterface
}

// NewProvider creates a new secret provider based on a secretsClient
func NewProvider(secretsClient corev1.SecretInterface, namespace string) secrets.SecretProvider {
	return &provider{
		namespace:     namespace,
		secretsClient: secretsClient,
	}
}

// GetSecret returns secret with the given name from the defined namespace if existing.
func (p *provider) GetSecret(name string) (*v1.Secret, error) {
	secret, err := p.secretsClient.Get(name, metav1.GetOptions{})
	if err != nil {
		errorWithMessage := errors.WithMessagef(err, "failed to get secret '%s' in namespace '%s'", name, p.namespace)
		log.Printf(errorWithMessage.Error())
		return secret, errorWithMessage
	}
	return secret, nil
}
