package k8s

import (
	"context"
	"time"

	stewardclients "github.com/SAP/stewardci-core/pkg/client/clientset/versioned"
	stewardv1alpha1client "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	stewardinformers "github.com/SAP/stewardci-core/pkg/client/informers/externalversions"
	tektonclients "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned"
	tektonv1beta1client "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/typed/pipeline/v1beta1"
	tektoninformers "github.com/SAP/stewardci-core/pkg/tektonclient/informers/externalversions"
	dynamic "k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingv1client "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog/v2"
)

// ClientFactory is the interface for Kubernetes client factories.
type ClientFactory interface {
	// CoreV1 returns the core/v1 Kubernetes client
	CoreV1() corev1client.CoreV1Interface

	// NetworkingV1 returns the networking/v1 Kubernetes client
	NetworkingV1() networkingv1client.NetworkingV1Interface

	// RbacV1 returns the rbac/v1 Kubernetes client
	RbacV1() rbacv1client.RbacV1Interface

	// Dynamic returns the dynamic Kubernetes client
	Dynamic() dynamic.Interface

	// StewardV1alpha1 returns the steward.sap.com/v1alpha1 Kubernetes client
	StewardV1alpha1() stewardv1alpha1client.StewardV1alpha1Interface

	// StewardInformerFactory returns the informer factory for Steward
	StewardInformerFactory() stewardinformers.SharedInformerFactory

	// TektonV1beta1 returns the tekton.dev/v1beta1 Kubernetes client
	TektonV1beta1() tektonv1beta1client.TektonV1beta1Interface

	// TektonInformerFactory returns the informer factory for Tekton
	TektonInformerFactory() tektoninformers.SharedInformerFactory
}

type clientFactory struct {
	kubernetesClientset    *kubernetes.Clientset
	dynamicClient          dynamic.Interface
	stewardClientset       *stewardclients.Clientset
	stewardInformerFactory stewardinformers.SharedInformerFactory
	tektonClientset        *tektonclients.Clientset
	tektonInformerFactory  tektoninformers.SharedInformerFactory
}

// NewClientFactory creates new client factory based on rest config
func NewClientFactory(config *rest.Config, resyncPeriod time.Duration) ClientFactory {
	logger := klog.FromContext(context.Background())

	stewardClientset, err := stewardclients.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create Steward clientset")
		return nil
	}
	stewardInformerFactory := stewardinformers.NewSharedInformerFactory(stewardClientset, resyncPeriod)

	kubernetesClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create Kubernetes clientset")
		return nil
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create dynamic Kubernetes clientset")
		return nil
	}

	tektonClientset, err := tektonclients.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed to create Tekton clientset")
		return nil
	}
	tektonInformerFactory := tektoninformers.NewSharedInformerFactory(tektonClientset, resyncPeriod)

	return &clientFactory{
		kubernetesClientset:    kubernetesClientset,
		dynamicClient:          dynamicClient,
		stewardClientset:       stewardClientset,
		stewardInformerFactory: stewardInformerFactory,
		tektonClientset:        tektonClientset,
		tektonInformerFactory:  tektonInformerFactory,
	}
}

// StewardInformerFactory implements interface ClientFactory
func (f *clientFactory) StewardInformerFactory() stewardinformers.SharedInformerFactory {
	return f.stewardInformerFactory
}

// StewardV1alpha1 implements interface ClientFactory
func (f *clientFactory) StewardV1alpha1() stewardv1alpha1client.StewardV1alpha1Interface {
	return f.stewardClientset.StewardV1alpha1()
}

// CoreV1 implements interface ClientFactory
func (f *clientFactory) CoreV1() corev1client.CoreV1Interface {
	return f.kubernetesClientset.CoreV1()
}

// Dynamic implements interface ClientFactory
func (f *clientFactory) Dynamic() dynamic.Interface {
	return f.dynamicClient
}

// NetworkingV1 implements interface ClientFactory
func (f *clientFactory) NetworkingV1() networkingv1client.NetworkingV1Interface {
	return f.kubernetesClientset.NetworkingV1()
}

// RbacV1 implements interface ClientFactory
func (f *clientFactory) RbacV1() rbacv1client.RbacV1Interface {
	return f.kubernetesClientset.RbacV1()
}

// TektonInformerFactory implements interface ClientFactory
func (f *clientFactory) TektonInformerFactory() tektoninformers.SharedInformerFactory {
	return f.tektonInformerFactory
}

// TektonV1beta1 implements interface ClientFactory
func (f *clientFactory) TektonV1beta1() tektonv1beta1client.TektonV1beta1Interface {
	return f.tektonClientset.TektonV1beta1()
}
