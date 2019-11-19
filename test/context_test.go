package test

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func Test_Context(t *testing.T) {
	ctx := context.Background()
	ctx = SetNamespace(ctx, "ns1")
	ctx = SetClientFactory(ctx, nil)
	assert.Equal(t, "ns1", GetNamespace(ctx))
}
