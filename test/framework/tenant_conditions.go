package framework

import (
	"context"
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
)

// TenantCheck is a Check for a tenant
type TenantCheck func(*api.Tenant) bool

// CreateTenantCondition creates a WaitCondition for a tenant with a dedicated Check
func CreateTenantCondition(tenant *api.Tenant, Check TenantCheck) WaitConditionFunc {
	key := fmt.Sprintf("%s/%s", tenant.GetNamespace(), tenant.GetName())
	return func(ctx context.Context) (bool, error) {
		fetcher := k8s.NewTenantFetcher(GetClientFactory(ctx))
		tenant, err := fetcher.ByKey(key)
		if err != nil {
			return true, err
		}
		return Check(tenant), nil
	}
}

// TenantHasStateResult creates a TenantCheck which Checks for a dedicated State
func TenantHasStateResult(result api.TenantResult) TenantCheck {
	return func(tenant *api.Tenant) bool {
		return tenant.Status.Result == result
	}
}
