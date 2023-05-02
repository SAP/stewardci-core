package k8s

import (
	"context"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/v3/assert"
)

const name string = "MyName"

func Test_contentNamespace_GetSecretProvider_works(t *testing.T) {
	// SETUP
	ctx := context.Background()

	cf := fake.NewClientFactory(
		fake.SecretOpaque(name, ns1),
	)
	examinee := NewContentNamespace(cf, ns1)

	// EXERCISE
	result := examinee.GetSecretProvider()

	// VERIFY
	storedSecret, err := result.GetSecret(ctx, name)
	assert.NilError(t, err)
	assert.Equal(t, name, storedSecret.GetName())
	assert.Equal(t, "", storedSecret.GetNamespace())
}
