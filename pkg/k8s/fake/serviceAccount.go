package fake

import (
	v1 "k8s.io/api/core/v1"
)

// ServiceAccount dummy
func ServiceAccount(name string, namespace string) *v1.ServiceAccount {
	return &v1.ServiceAccount{ObjectMeta: ObjectMeta(name, namespace)}
}
