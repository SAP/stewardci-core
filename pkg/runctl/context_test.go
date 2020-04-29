package runctl

import (
	"context"
	"gotest.tools/assert"
	"testing"
)

func Test_getRunInstanceTesting_returns_nil_if_not_set(t *testing.T) {
	ctx := context.Background()
	assert.Assert(t, getRunInstanceTesting(ctx) == nil)
}

func Test_setgetRunInstanceTesting(t *testing.T) {
	ctx := context.Background()
	rit := &runInstanceTesting{}
	ctx = withRunInstanceTesting(ctx, rit)
	assert.Assert(t, rit == getRunInstanceTesting(ctx))
}
