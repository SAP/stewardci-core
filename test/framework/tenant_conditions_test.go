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
	corev1 "k8s.io/api/core/v1"
	knativeapis "knative.dev/pkg/apis"
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

func Test_TenantIsReady(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		condition      *knativeapis.Condition
		expectedResult bool
	}{{
		condition: &knativeapis.Condition{
			Type:   knativeapis.ConditionReady,
			Status: corev1.ConditionTrue,
		},
		expectedResult: true,
	}, {
		condition:      nil,
		expectedResult: false,
	}, {
		condition: &knativeapis.Condition{
			Type:   knativeapis.ConditionReady,
			Status: corev1.ConditionFalse,
		},
		expectedResult: false,
	},
	} {
		// SETUP
		examine := TenantIsReady()
		tenant := builder.TenantFixName("foo", "bar")
		if test.condition != nil {
			tenant.Status.SetCondition(test.condition)
		}
		// EXERCISE
		result := examine(tenant)
		// VERIFY
		assert.Assert(t, result == test.expectedResult)
	}
}
