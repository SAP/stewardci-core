package fake

import (
	"context"

	utils "github.com/SAP/stewardci-core/pkg/utils"
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
	logger := utils.LoggerFromContext(context.Background())
	meta := ObjectMeta(name, namespace)
	result, err := cache.MetaNamespaceKeyFunc(&meta)
	if err != nil {
		logger.Error(err, "Failed to extract object metadata",
			"object", klog.KObj(&meta),
		)
	}
	return result
}
