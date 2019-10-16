package k8s

import (
	"fmt"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
)

const name string = "MyName"

func Test_GetSecret_works(t *testing.T) {
	factory := fake.NewClientFactory(fake.Secret(name, ns1))
	tn := NewTenantNamespace(factory, ns1)
	storedSecret, _ := tn.GetSecret(name)
	assert.Equal(t, name, storedSecret.GetName(), "Name should be equal")
	assert.Equal(t, ns1, storedSecret.GetNamespace(), "Namespace should be equal")
	assert.Equal(t, fake.SecretValue, storedSecret.StringData[fake.SecretKey])
}

func Test__GetSecret__failsWithMissingSecret(t *testing.T) {
	factory := fake.NewClientFactory()
	tn := NewTenantNamespace(factory, ns1)
	_, err := tn.GetSecret(name)
	expectedMessage := fmt.Sprintf("Failed to get secret '%s' in namespace '%s': secrets \"%s\" not found",
		name, ns1, name)
	assert.Equal(t, expectedMessage, err.Error())
}
