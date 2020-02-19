package framework

import (
	"context"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
)

func Test_Set_GetClientFactory(t *testing.T) {
	ctx := context.Background()
	factory := fake.NewClientFactory()
	ctx = SetClientFactory(ctx, factory)
	assert.Assert(t, factory == GetClientFactory(ctx))
}

func Test_Set_GetNamespace(t *testing.T) {
	ctx := context.Background()
	ctx = SetNamespace(ctx, "ns1")
	assert.Equal(t, "ns1", GetNamespace(ctx))
}

func Test_Set_GetTenantNamespace(t *testing.T) {
	ctx := context.Background()
	ctx = SetTenantNamespace(ctx, "ns1")
	assert.Equal(t, "ns1", GetTenantNamespace(ctx))
}

func Test_GetTenantNamespace_defaultsToEmptyString(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "", GetTenantNamespace(ctx))
}

func Test_Set_GetTestName(t *testing.T) {
	ctx := context.Background()
	ctx = SetTestName(ctx, "foo")
	assert.Equal(t, "foo", GetTestName(ctx))
}

func Test_Set_GetPipelineRun(t *testing.T) {
	ctx := context.Background()
	run := builder.PipelineRun("foo", "bar")
	ctx = SetPipelineRun(ctx, run)
	assert.DeepEqual(t, run, GetPipelineRun(ctx))
}

func Test_Set_GetRealmUUID(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "", GetRealmUUID(ctx))

	ctx = SetRealmUUID(ctx)
	assert.Assert(t, GetRealmUUID(ctx) != "")
}
