package k8s

import (
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	"testing"
)

const name string = "MyName"

func Test_GetSecret_works(t *testing.T) {
	factory := fake.NewClientFactory(fake.Secret(name, ns1))
	tn := NewTenantNamespace(factory, ns1)
	storedSecret, _ := tn.GetSecretProvider().GetSecret(name)
	assert.Equal(t, name, storedSecret.GetName(), "Name should be equal")
	assert.Equal(t, ns1, storedSecret.GetNamespace(), "Namespace should be equal")
	assert.Equal(t, fake.SecretValue, storedSecret.StringData[fake.SecretKey])
}
