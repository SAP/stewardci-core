package builder

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Tenant(name, namespace, displayName string) {
	t := &api.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name: name,
		},
		Spec: api.TenantSpec{
			Name: name
			DisplayName: displayName
		}
	}
	return t
}