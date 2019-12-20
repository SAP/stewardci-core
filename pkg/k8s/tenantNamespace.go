package k8s

import (
	"log"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	k8ssecretprovider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/k8s"
)

// TenantNamespace representing the client
type TenantNamespace interface {
	GetSecretProvider() secrets.SecretProvider
	TargetClientFactory() ClientFactory
}

type tenantNamespace struct {
	pipelineRunClient stewardv1alpha1.PipelineRunInterface
	factory           ClientFactory
	secretProvider    secrets.SecretProvider
}

// NewTenantNamespace creates new TenantNamespace object
func NewTenantNamespace(factory ClientFactory, namespace string) TenantNamespace {
	secretsClient := factory.CoreV1().Secrets(namespace)
	pipelineRunClient := factory.StewardV1alpha1().PipelineRuns(namespace)
	secretProvider := k8ssecretprovider.NewProvider(secretsClient, namespace)
	return &tenantNamespace{
		secretProvider:    secretProvider,
		pipelineRunClient: pipelineRunClient,
		factory:           factory,
	}
}

// TargetClientFactory returns ClientFactory for workload
func (t *tenantNamespace) TargetClientFactory() ClientFactory {
	return t.factory
}

//  GetSecretProvider returns a secret provider
func (t *tenantNamespace) GetSecretProvider() secrets.SecretProvider {
	return t.secretProvider
}
