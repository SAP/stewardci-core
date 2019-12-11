// +build e2e

package schemavalidationtests

import (
	"testing"

	framework "github.com/SAP/stewardci-core/test/framework"
	"gotest.tools/assert"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func Test_TenantSchemaValidation(t *testing.T) {
	for _, test := range tenantTests {
		t.Run(test.name, func(t *testing.T) {
			// PREPARE
			ctx := framework.Setup(t)

			// EXERCISE
			tenant, err := framework.CreateTenantFromJSON(ctx, test.data)
			defer framework.DeleteTenant(ctx, tenant)

			// VERIFY
			test.check(t, err)
		})
	}
}

var tenantTests = []SchemaValidationTest{

	// ###################################################################
	SchemaValidationTest{
		name:       "good case with generateName",
		dataFormat: json,
		data: `
		{
			"apiVersion": "steward.sap.com/v1alpha1",
			"kind": "Tenant",
			"metadata": {
				"generateName": "test-tenant-validation-"
			}
		}`,
		check: func(t *testing.T, err error) {
			assert.NilError(t, err)
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "good case with name",
		dataFormat: json,
		data: `
		{
			"apiVersion": "steward.sap.com/v1alpha1",
			"kind": "Tenant",
			"metadata": {
				"name": "tenant1"
			}
		}`,
		check: func(t *testing.T, err error) {
			assert.NilError(t, err)
		},
	},
}
