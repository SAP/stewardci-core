package framework

import (
	"context"
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	knativeapis "knative.dev/pkg/apis"
)

// TenantCheck is a Check for a tenant
type TenantCheck func(*api.Tenant) bool

// CreateTenantCondition creates a WaitCondition for a tenant with a dedicated Check
func CreateTenantCondition(tenant *api.Tenant, Check TenantCheck) WaitConditionFunc {
	key := fmt.Sprintf("%s/%s", tenant.GetNamespace(), tenant.GetName())
	return func(ctx context.Context) (bool, error) {
		fetcher := k8s.NewClientBasedTenantFetcher(GetClientFactory(ctx))
		tenant, err := fetcher.ByKey(ctx, key)
		if err != nil {
			return true, err
		}
		if tenant == nil {
			return true, fmt.Errorf("tenant not found %q", key)
		}
		return Check(tenant), nil
	}
}

// TenantIsReady creates a TenantCheck which Checks if tenant has readyCondition wiht status true
func TenantIsReady() TenantCheck {
	return func(tenant *api.Tenant) bool {
		readyCondition := tenant.Status.GetCondition(knativeapis.ConditionReady)
		if readyCondition == nil {
			return false
		}
		return readyCondition.Status == corev1.ConditionTrue
	}
}
