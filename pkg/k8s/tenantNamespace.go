package k8s

import (
	"log"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	errors "github.com/SAP/stewardci-core/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// TenantNamespace representing the client
type TenantNamespace interface {
	SecretProvider
	TargetClientFactory() ClientFactory
}

// SecretProvider provides secrets
type SecretProvider interface {
	GetSecret(name string) (*v1.Secret, error)
}

type tenantNamespace struct {
	namespace         string
	secretName        string
	secretsClient     corev1.SecretInterface
	pipelineRunClient stewardv1alpha1.PipelineRunInterface
	factory           ClientFactory
}

// NewTenantNamespace creates new TenantNamespace object
func NewTenantNamespace(factory ClientFactory, namespace string) TenantNamespace {
	secretsClient := factory.CoreV1().Secrets(namespace)
	pipelineRunClient := factory.StewardV1alpha1().PipelineRuns(namespace)
	log.Printf("Creating tenantNamespace: '%s'", namespace)
	return &tenantNamespace{
		namespace:         namespace,
		secretsClient:     secretsClient,
		pipelineRunClient: pipelineRunClient,
		factory:           factory,
	}
}

// TargetClientFactory returns ClientFactory for workload
func (t *tenantNamespace) TargetClientFactory() ClientFactory {
	return t.factory
}

//  GetSecret returns secret
func (t *tenantNamespace) GetSecret(name string) (*v1.Secret, error) {
	secret, err := t.secretsClient.Get(name, metav1.GetOptions{})
	if err != nil {
		errorWithMessage := errors.Errorf(err, "Failed to get secret '%s' in namespace '%s'", name, t.namespace)
		log.Printf(errorWithMessage.Error())
		return secret, errorWithMessage
	}
	return secret, nil
}
