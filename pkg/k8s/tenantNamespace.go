package k8s

import (
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	provider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/k8s"
	"log"
)

// TenantNamespace representing the client
type TenantNamespace interface {
	GetSecretProvider() secrets.SecretProvider
	TargetClientFactory() ClientFactory
}

type tenantNamespace struct {
	pipelineRunClient stewardv1alpha1.PipelineRunInterface
	factory           ClientFactory
	provider          secrets.SecretProvider
}

// NewTenantNamespace creates new TenantNamespace object
func NewTenantNamespace(factory ClientFactory, namespace string) TenantNamespace {
	secretsClient := factory.CoreV1().Secrets(namespace)
	pipelineRunClient := factory.StewardV1alpha1().PipelineRuns(namespace)
	provider := provider.NewProvider(secretsClient, namespace)
	log.Printf("Creating tenantNamespace: '%s'", namespace)

	return &tenantNamespace{
		provider:          provider,
		pipelineRunClient: pipelineRunClient,
		factory:           factory,
	}
}

// TargetClientFactory returns ClientFactory for workload
func (t *tenantNamespace) TargetClientFactory() ClientFactory {
	return t.factory
}

//  GetSecret returns secret
func (t *tenantNamespace) GetSecretProvider() secrets.SecretProvider {
	return t.provider
}
