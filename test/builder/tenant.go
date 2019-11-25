package builder

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tenant creates a Tenant
func Tenant(name, namespace, displayName string) *api.Tenant {
	t := &api.Tenant{
                ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "t-",
		},
		Spec: api.TenantSpec{
			Name:        name,
			DisplayName: displayName,
		},
	}
	return t
}

// TenantFixName creates a Tenant with a fixed name
func TenantFixName(name, namespace string) *api.Tenant {
        t := &api.Tenant{
                ObjectMeta: metav1.ObjectMeta{
                        Namespace:    namespace,
                        Name: name,
                },
        }
        return t
}

