package k8s

import (
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"testing"
)

func Test_provider_GetSecret_Existing(t *testing.T) {
	namespace := "ns1"
	cf := fake.NewClientFactory(fake.Secret("foo", namespace))
	secretsClient := cf.CoreV1().Secrets(namespace)
	provider := NewProvider(secretsClient, namespace)
	secret, err := provider.GetSecret("foo")
	assert.NilError(t, err)
	assert.Equal(t, "foo", secret.GetName())
}

func Test_provider_GetSecret_NotExisting(t *testing.T) {
	namespace := "ns1"
	cf := fake.NewClientFactory(fake.Secret("foo", namespace))
	secretsClient := cf.CoreV1().Secrets(namespace)
	provider := NewProvider(secretsClient, namespace)
	secret, err := provider.GetSecret("bar")
	assert.Assert(t, is.Regexp("failed to get secret 'bar' in namespace 'ns1'.+", err.Error()))
	assert.Assert(t, secret == nil)
}
