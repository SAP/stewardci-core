package test

import (
    "fmt"

api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
"github.com/SAP/stewardci-core/pkg/k8s"
)


type WaitCondition func(k8s.ClientFactory) (bool, error)

type TenantStateCheck func(*api.Tenant) (bool) 

func CreateTenantCondition(tenant *api.Tenant, check TenantStateCheck) WaitCondition {
  return func (clientFactory k8s.ClientFactory) (bool, error) {
      fetcher := k8s.NewTenantFetcher(clientFactory)
      key := fmt.Sprintf("%s/%s",tenant.GetNamespace(),tenant.GetName())
      tenant,err := fetcher.ByKey(key)
      if err != nil {
          return true,err
      }
      return check(tenant),nil
}
}

func TenantHasStateResult(result api.TenantResult) TenantStateCheck {
    return  func(tenant *api.Tenant) (bool) {
   return tenant.Status.Result == result
}
} 
