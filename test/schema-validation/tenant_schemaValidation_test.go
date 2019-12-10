package schema-validation

import (
	"fmt"
	"testing"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	framework "github.com/SAP/stewardci-core/test/framework"
	"gotest.tools/assert"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func Test_TenantSchemaValidation(t *testing.T) {
	data, checks := getTenantSchemaTestData()

	for testName, testJSON := range data {
		t.Run(testName, func(t *testing.T) {
			// PREPARE
			ctx := framework.Setup(t)

			// EXERCISE
			tenant, err := framework.CreateTenantFromJSON(ctx, testJSON)
			defer framework.DeleteTenant(ctx, tenant)

			// VERIFY
			check := checks[testName]
			check.(func(t *testing.T, tenant *v1alpha1.Tenant, err error))(t, tenant, err)
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

func getTenantSchemaTestData() (data map[string]string, checks map[string]interface{}) {
	data = map[string]string{}
	checks = map[string]interface{}{}
	var testName string

	// good case #################
	testName = "good case"
	data[testName] = fmt.Sprintf(`
		{
			%v,
			"spec": {
				"name": "name1",
				"displayName": "displayName1"
			}
		}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.NilError(t, err)
	}

	// spec #################
	testName = "spec empty"
	data[testName] = fmt.Sprintf(`
	{
		%v,
		"spec": {}
	}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.NilError(t, err)
	}

	testName = "spec missing"
	data[testName] = `
	{
		"apiVersion": "steward.sap.com/v1alpha1",
		"kind": "Tenant",
		"metadata": {
			"name": "tenant1"
		}
	}`
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.NilError(t, err)
	}

	// spec.name #################
	testName = "spec.name missing"
	data[testName] = fmt.Sprintf(`
	{
		%v,
		"spec": {
			"displayName": "displayName1"
		}
	}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.NilError(t, err)
	}

	testName = "spec.name empty"
	data[testName] = fmt.Sprintf(`
	{
		%v,
		"spec": {
			"name": "",
			"displayName": "displayName1"
		}
	}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.ErrorContains(t, err, "spec.name in body should match '^[^\\s]{1,}.*$'")
	}

	testName = "spec.name is number"
	data[testName] = fmt.Sprintf(`
	{
		%v,
		"spec": {
			"name": 1,
			"displayName": "displayName1"
		}
	}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.ErrorContains(t, err, "spec.name in body must be of type string: \"integer\"")
	}

	// spec.displayName #################
	testName = "spec.displayName missing"
	data[testName] = fmt.Sprintf(`
	{
		%v,
		"spec": {
			"name": "name1"
		}
	}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.NilError(t, err)
	}

	testName = "spec.displayName empty"
	data[testName] = fmt.Sprintf(`
	{
		%v,
		"spec": {
			"name": "name1",
			"displayName": ""
		}
	}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.ErrorContains(t, err, "spec.displayName in body should match '^[^\\s]{1,}.*$'")
	}

	testName = "spec.displayName is number"
	data[testName] = fmt.Sprintf(`
	{
		%v,
		"spec": {
			"name": "name1",
			"displayName": 1
		}
	}`, tenantHeader)
	checks[testName] = func(t *testing.T, tenant *v1alpha1.Tenant, err error) {
		assert.ErrorContains(t, err, "spec.displayName in body must be of type string: \"integer\"")
	}

	return
}
