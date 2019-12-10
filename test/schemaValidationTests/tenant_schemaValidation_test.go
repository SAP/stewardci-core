// +build e2e

package schemavalidationtests

import (
	"fmt"
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

const tenantHeader string = `
"apiVersion": "steward.sap.com/v1alpha1",
"kind": "Tenant",
"metadata": {
	"generateName": "test-tenant-validation-"
}
`

var tenantTests = []SchemaValidationTest{

	// ###################################################################
	SchemaValidationTest{
		name:       "good case",
		dataFormat: json,
		data: fmt.Sprintf(`
		{
			%v,
			"spec": {
				"name": "name1",
				"displayName": "displayName1"
			}
		}`, tenantHeader),
		check: func(t *testing.T, err error) {
			assert.NilError(t, err)
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec empty",
		dataFormat: json,
		data: fmt.Sprintf(`
		{
			%v,
			"spec": {}
		}`, tenantHeader),
		check: func(t *testing.T, err error) {
			assert.NilError(t, err)
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec missing",
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

	// ###################################################################
	SchemaValidationTest{
		name:       "spec values are empty",
		dataFormat: json,
		data: fmt.Sprintf(`
		{
			%v,
			"spec": {
				"name": "",
				"displayName": ""
			}
		}`, tenantHeader),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "spec.name in body should match '^[^\\s]{1,}.*$'")
			assert.ErrorContains(t, err, "spec.displayName in body should match '^[^\\s]{1,}.*$'")
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec values are number",
		dataFormat: json,
		data: fmt.Sprintf(`
		{
			%v,
			"spec": {
				"name": 1,
				"displayName": 1
			}
		}`, tenantHeader),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "spec.name in body must be of type string: \"integer\"")
			assert.ErrorContains(t, err, "spec.displayName in body must be of type string: \"integer\"")
		},
	},
}
