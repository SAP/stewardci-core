package framework

import (
	"context"
	"fmt"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

// TODO: Check with clientFactory returning error on get
func Test_CreateTenantCondition(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		found          bool
		checkResult    bool
		expectedResult bool
		error          string
	}{
		{true, true, true, ""},
		{true, false, false, ""},
		{false, true, true, "tenant not found .*"},
		{false, false, true, "tenant not found .*"},
	} {
		name := fmt.Sprintf("Found: %t, check: %t", test.found, test.checkResult)
		t.Run(name, func(t *testing.T) {
			// SETUP
			ctx := context.Background()
			tenant := builder.TenantFixName("foo", "bar")
			clientFactory := fake.NewClientFactory()
			if test.found {
				_, err := clientFactory.StewardV1alpha1().Tenants("bar").Create(tenant)
				assert.NilError(t, err, "Setup error")
			}
			ctx = SetClientFactory(ctx, clientFactory)
			check := func(*api.Tenant) bool {
				return test.checkResult
			}
			// EXERCISE
			condition := CreateTenantCondition(tenant, check)
			result, err := condition(ctx)
			// VERIFY
			if test.error == "" {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, err != nil)
				assert.Assert(t, is.Regexp(test.error, err.Error()))
			}
			assert.Assert(t, test.expectedResult == result)
		})
	}
}

func Test_TenantHasStateResult(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		tenant         *api.Tenant
		desiredResult  api.TenantResult
		expectedResult bool
	}{{
		tenant:         &api.Tenant{Status: api.TenantStatus{Result: api.TenantResultSuccess}},
		desiredResult:  api.TenantResultSuccess,
		expectedResult: true,
	}, {
		tenant:         &api.Tenant{Status: api.TenantStatus{Result: api.TenantResultSuccess}},
		desiredResult:  api.TenantResultErrorInfra,
		expectedResult: false,
	}, {
		tenant:         &api.Tenant{Status: api.TenantStatus{Result: api.TenantResultErrorInfra}},
		desiredResult:  api.TenantResultSuccess,
		expectedResult: false,
	}, {
		tenant:         &api.Tenant{Status: api.TenantStatus{Result: api.TenantResultErrorInfra}},
		desiredResult:  api.TenantResultErrorInfra,
		expectedResult: true,
	},
	} {
		// SETUP
		examine := TenantHasStateResult(test.desiredResult)
		// EXERCISE
		result := examine(test.tenant)
		// VERIFY
		assert.Assert(t, result == test.expectedResult)
	}
}
