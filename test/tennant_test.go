// +build e2e

package test

import (
	"testing"

"gotest.tools/assert"
	"github.com/SAP/stewardci-core/test/builder"
)

func TestTenantCreation(t *testing.T) {
	t.Parallel()
	clientFactory, namespace := setup(t)
	tenant := builder.Tenant("name", namespace, "displayName")
	stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	_, err := stewardClient.Create(tenant)
        assert.NilError(t,err)
}
