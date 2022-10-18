//go:build e2e
// +build e2e

package tenant

import (
	"testing"

	framework "github.com/SAP/stewardci-core/test/framework"
	"gotest.tools/v3/assert"
)

func Test_CRDs_Tenant_Schema(t *testing.T) {
	testcases := []struct {
		name     string
		manifest string
		check    func(t *testing.T, resultErr error)
	}{

		{
			name: "good case with generateName",
			manifest: `
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "Tenant",
					"metadata": {
						"generateName": "test-tenant-validation-",
						"labels": {
							"steward.sap.com/ignore": ""
						}
					}
				}
			`,
			check: func(t *testing.T, err error) {
				assert.NilError(t, err)
			},
		},

		{
			name: "good case with name",
			manifest: `
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "Tenant",
					"metadata": {
						"name": "tenant1",
						"labels": {
							"steward.sap.com/ignore": ""
						}
					}
				}
			`,
			check: func(t *testing.T, err error) {
				assert.NilError(t, err)
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			testcase := testcase
			t.Parallel()

			// SETUP
			ctx := framework.Setup(t)

			// EXERCISE
			tenant, err := framework.CreateTenantFromJSON(ctx, testcase.manifest)
			if tenant != nil {
				defer framework.DeleteTenant(ctx, tenant)
			}

			// VERIFY
			testcase.check(t, err)
		})
	}
}
