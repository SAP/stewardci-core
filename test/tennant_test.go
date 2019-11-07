// +build e2e

package test

import (
	"log"
	"testing"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/test/builder"
	knativetest "knative.dev/pkg/test"
)


func TestTenantCreation(t *testing.T) {
	t.Parallel()
	clientFactory,namespace := setup(t)
	tenant := builder.Tenant("name",namespace,"displayName")
	stewardClient := clientFactory.StewardV1alpha1().Tenants(tenant.GetNamespace())
	stewardClient.Create(tenant)
}