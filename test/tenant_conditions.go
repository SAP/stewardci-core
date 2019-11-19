package test

import (
	"context"
	"fmt"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
)

// TenantCheck is a check for a tenant
type TenantCheck func(*api.Tenant) bool

// CreateTenantCondition creates a WaitCondition for a tenant with a dedicated check
func CreateTenantCondition(tenant *api.Tenant, check TenantCheck) WaitCondition {
	key := fmt.Sprintf("%s/%s", tenant.GetNamespace(), tenant.GetName())
	return NewWaitCondition(func(ctx context.Context) (bool, error) {
		fetcher := k8s.NewTenantFetcher(GetClientFactory(ctx))
		tenant, err := fetcher.ByKey(key)
		if err != nil {
			return true, err
		}
		return check(tenant), nil
	})
}

// TenantHasStateResult creates a TenantCheck which checks for a dedicated State
func TenantHasStateResult(result api.TenantResult) TenantCheck {
	return func(tenant *api.Tenant) bool {
		return tenant.Status.Result == result
	}
}
