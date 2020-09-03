package fake

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	klog "k8s.io/klog/v2"
)

// ObjectMeta returns a fake ObjectMeta with a given name and namespace
func ObjectMeta(name string, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: namespace}
}

// ObjectKey returns a fake key string with a given name and namespace
func ObjectKey(name string, namespace string) string {
	meta := ObjectMeta(name, namespace)
	result, err := cache.MetaNamespaceKeyFunc(&meta)
	if err != nil {
		klog.V(2).Printf("Error creating key: %s", err.Error())
	}
	return result
}
