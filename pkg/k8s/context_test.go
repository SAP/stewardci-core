package k8s

import (
	"context"
	"gotest.tools/assert"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
)

func Test_SetGetNamespaceManager(t *testing.T) {
	ctx := context.Background()
	factory := fake.NewClientFactory()
	namespaceManager := NewNamespaceManager(factory, "", 0)
	ctx = WithNamespaceManager(ctx, namespaceManager)
	assert.Assert(t, namespaceManager == GetNamespaceManager(ctx))

}

func Test_SetGetClientFactory(t *testing.T) {
	ctx := context.Background()
	factory := fake.NewClientFactory()
	ctx = WithClientFactory(ctx, factory)
	assert.Assert(t, factory == GetClientFactory(ctx))
}

func Test_SetGetServiceAccountTokenSecretRetriever(t *testing.T) {
	ctx := context.Background()
	sac := &serviceAccountTokenSecretRetrieverImpl{}
	ctx = WithServiceAccountTokenSecretRetriever(ctx, sac)
	assert.Assert(t, sac == GetServiceAccountTokenSecretRetriever(ctx))
}
