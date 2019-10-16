package fake

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Namespace creates a fake namespace with defined name.
func Namespace(name string) *v1.Namespace {
	return &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

// NamespaceWithAnnotations creates a fake namespace with defined name and annotations.
func NamespaceWithAnnotations(name string, annotations map[string]string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
	}
}
