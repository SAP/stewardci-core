package fake

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tenant creates a new fake tenant object.
func Tenant(tenantID, tenantName, displayName, namespace string) *api.Tenant {
	typeMeta := metav1.TypeMeta{Kind: "Tenant", APIVersion: "steward.sap.com/v1alpha1"}
	objectMeta := metav1.ObjectMeta{Name: tenantID, Namespace: namespace}
	return &api.Tenant{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec: api.TenantSpec{
			Name:        tenantName,
			DisplayName: displayName,
		},
	}
}
