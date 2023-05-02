package k8s

import (
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	secrets "github.com/SAP/stewardci-core/pkg/k8s/secrets"
	k8ssecretprovider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/k8s"
)

// ContentNamespace representing the client
type ContentNamespace interface {
	GetSecretProvider() secrets.SecretProvider
	TargetClientFactory() ClientFactory
}

type contentNamespace struct {
	pipelineRunClient stewardv1alpha1.PipelineRunInterface
	factory           ClientFactory
	secretProvider    secrets.SecretProvider
}

// NewContentNamespace creates new ContentNamespace object
func NewContentNamespace(factory ClientFactory, namespace string) ContentNamespace {
	secretsClient := factory.CoreV1().Secrets(namespace)
	pipelineRunClient := factory.StewardV1alpha1().PipelineRuns(namespace)
	secretProvider := k8ssecretprovider.NewProvider(secretsClient, namespace)
	return &contentNamespace{
		secretProvider:    secretProvider,
		pipelineRunClient: pipelineRunClient,
		factory:           factory,
	}
}

// TargetClientFactory returns ClientFactory for workload
func (t *contentNamespace) TargetClientFactory() ClientFactory {
	return t.factory
}

//  GetSecretProvider returns a secret provider
func (t *contentNamespace) GetSecretProvider() secrets.SecretProvider {
	return t.secretProvider
}
