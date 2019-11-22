package framework

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

func Test_Set_GetTestName(t *testing.T) {
	ctx := context.Background()
	ctx = SetTestName(ctx, "foo")
	assert.Equal(t, "foo", GetTestName(ctx))
}
