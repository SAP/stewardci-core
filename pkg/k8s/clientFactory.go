package k8s

import (
	"time"

	steward "github.com/SAP/stewardci-core/pkg/client/clientset/versioned"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	stewardinformer "github.com/SAP/stewardci-core/pkg/client/informers/externalversions"
	tektonclient "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned"
	tektonclientv1alpha1 "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/typed/pipeline/v1alpha1"
	tektoninformers "github.com/SAP/stewardci-core/pkg/tektonclient/informers/externalversions"
	dynamic "k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog/v2"
)

// ClientFactory is the interface for Kubernet client factories.
type ClientFactory interface {
	// CoreV1 returns the core/v1 Kubernetes client
	CoreV1() corev1.CoreV1Interface

	// NetworkingV1 returns the networking/v1 Kubernetes client
	NetworkingV1() networkingv1.NetworkingV1Interface

	// RbacV1beta1 returns the rbac/v1beta1 Kubernetes client
	RbacV1beta1() rbacv1beta1.RbacV1beta1Interface

	// Dynamic returns the dynamic Kubernetes client
	Dynamic() dynamic.Interface

	// StewardV1alpha1 returns the steward.sap.com/v1alpha1 Kubernetes client
	StewardV1alpha1() stewardv1alpha1.StewardV1alpha1Interface

	// StewardInformerFactory returns the informer factory for Steward
	StewardInformerFactory() stewardinformer.SharedInformerFactory

	// TektonV1alpha1 returns the tekton.dev/v1alpha1 Kubernetes client
	TektonV1alpha1() tektonclientv1alpha1.TektonV1alpha1Interface

	// TektonInformerFactory returns the informer factory for Tekton
	TektonInformerFactory() tektoninformers.SharedInformerFactory
}

type clientFactory struct {
	kubernetesClientset    *kubernetes.Clientset
	dynamicClient          dynamic.Interface
	stewardClientset       *steward.Clientset
	stewardInformerFactory stewardinformer.SharedInformerFactory
	tektonClientset        *tektonclient.Clientset
	tektonInformerFactory  tektoninformers.SharedInformerFactory
}

// NewClientFactory creates new client factory based on rest config
func NewClientFactory(config *rest.Config, resyncPeriod time.Duration) ClientFactory {
	stewardClientset, err := steward.NewForConfig(config)
	if err != nil {
		klog.V(2).Printf("could not create Steward clientset: %s", err)
		return nil
	}
	stewardInformerFactory := stewardinformer.NewSharedInformerFactory(stewardClientset, resyncPeriod)

	kubernetesClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.V(2).Printf("could not create Kubernetes clientset: %s", err)
		return nil
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.V(2).Printf("could not create dynamic Kubernetes clientset: %s", err)
		return nil
	}

	tektonClientset, err := tektonclient.NewForConfig(config)
	if err != nil {
		klog.V(2).Printf("could not create Tekton clientset: %s", err)
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
func (f *clientFactory) StewardInformerFactory() stewardinformer.SharedInformerFactory {
	return f.stewardInformerFactory
}

// StewardV1alpha1 implements interface ClientFactory
func (f *clientFactory) StewardV1alpha1() stewardv1alpha1.StewardV1alpha1Interface {
	return f.stewardClientset.StewardV1alpha1()
}

// CoreV1 implements interface ClientFactory
func (f *clientFactory) CoreV1() corev1.CoreV1Interface {
	return f.kubernetesClientset.CoreV1()
}

// Dynamic implements interface ClientFactory
func (f *clientFactory) Dynamic() dynamic.Interface {
	return f.dynamicClient
}

// NetworkingV1 implements interface ClientFactory
func (f *clientFactory) NetworkingV1() networkingv1.NetworkingV1Interface {
	return f.kubernetesClientset.NetworkingV1()
}

// RbacV1beta1 implements interface ClientFactory
func (f *clientFactory) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface {
	return f.kubernetesClientset.RbacV1beta1()
}

// TektonInformerFactory implements interface ClientFactory
func (f *clientFactory) TektonInformerFactory() tektoninformers.SharedInformerFactory {
	return f.tektonInformerFactory
}

// TektonV1alpha1 implements interface ClientFactory
func (f *clientFactory) TektonV1alpha1() tektonclientv1alpha1.TektonV1alpha1Interface {
	return f.tektonClientset.TektonV1alpha1()
}
