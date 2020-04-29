package secrets

import (
	"context"
	"gotest.tools/assert"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/fake"
)

func Test_GetSecretProvider_returns_nil_if_not_set(t *testing.T) {
	ctx := context.Background()
	assert.Assert(t, GetSecretProvider(ctx) == nil)
}

func Test_SetGetSecretProvider(t *testing.T) {
	ctx := context.Background()
	sp := fake.NewProvider("foo")
	ctx = WithSecretProvider(ctx, sp)
	assert.Assert(t, sp == GetSecretProvider(ctx))
}
