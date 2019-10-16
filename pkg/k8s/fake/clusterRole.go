package fake

import (
	v1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterRole creates a fake ClusterRole with defined name
func ClusterRole(name string) *v1beta1.ClusterRole {
	return &v1beta1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: name}}
}
