package fake

import (
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// ObjectMeta returns a fake ObjectMeta with a given name and namespace
func ObjectMeta(name string, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: namespace}
}

// ObjectMetaFull returns a fake ObjectMeta with a given name and namespace and dummy values
func ObjectMetaFull(name string, namespace string) metav1.ObjectMeta {
	now := metav1.Now()
	var grace int64 = 1
	return metav1.ObjectMeta{Name: name,
		GenerateName:               "dummy",
		Namespace:                  namespace,
		SelfLink:                   "dummy",
		UID:                        types.UID("dummy"),
		ResourceVersion:            "dummy",
		Generation:                 1,
		CreationTimestamp:          now,
		DeletionGracePeriodSeconds: &grace,
		OwnerReferences:            []metav1.OwnerReference{metav1.OwnerReference{}},
		Finalizers:                 []string{"dummy"},
		ClusterName:                "dummy",
	}

}

// ObjectKey returns a fake key string with a given name and namespace
func ObjectKey(name string, namespace string) string {
	meta := ObjectMeta(name, namespace)
	result, err := cache.MetaNamespaceKeyFunc(&meta)
	if err != nil {
		log.Printf("Error creating key: %s", err.Error())
	}
	return result
}
