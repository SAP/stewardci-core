package k8s

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
)

const name string = "MyName"

func Test_GetSecretProvider_works(t *testing.T) {
	factory := fake.NewClientFactory(fake.SecretOpaque(name, ns1))
	tn := NewTenantNamespace(factory, ns1)
	storedSecret, _ := tn.GetSecretProvider().GetSecret(name)
	assert.Equal(t, name, storedSecret.GetName())
	assert.Equal(t, "", storedSecret.GetNamespace())
}
