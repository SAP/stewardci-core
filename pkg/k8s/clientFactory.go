package k8s

import (
	"log"
	"time"

	steward "github.com/SAP/stewardci-core/pkg/client/clientset/versioned"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	stewardinformer "github.com/SAP/stewardci-core/pkg/client/informers/externalversions"
	tektonclient "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned"
	tektonclientv1alpha1 "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/typed/pipeline/v1alpha1"
	tektoninformers "github.com/SAP/stewardci-core/pkg/tektonclient/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	"k8s.io/client-go/rest"
)

// ClientFactory object
type clientFactory struct {
	kubernetesClientset    *kubernetes.Clientset
	stewardClientset       *steward.Clientset
	stewardInformerFactory stewardinformer.SharedInformerFactory
	tektonClientset        *tektonclient.Clientset
	tektonInformerFactory  tektoninformers.SharedInformerFactory
}

// ClientFactory interface
type ClientFactory interface {
	CoreV1() corev1.CoreV1Interface
	RbacV1beta1() rbacv1beta1.RbacV1beta1Interface
	StewardV1alpha1() stewardv1alpha1.StewardV1alpha1Interface
	StewardInformerFactory() stewardinformer.SharedInformerFactory
	TektonV1alpha1() tektonclientv1alpha1.TektonV1alpha1Interface
	TektonInformerFactory() tektoninformers.SharedInformerFactory
}

// NewClientFactory creates new client factory based on rest config
func NewClientFactory(config *rest.Config, resyncPeriod time.Duration) ClientFactory {
	stewardClientset, err := steward.NewForConfig(config)
	if err != nil {
		log.Printf("Cannot create steward client %s", err)
		return nil
	}

	stewardInformerFactory := stewardinformer.NewSharedInformerFactory(stewardClientset, resyncPeriod)

	kubernetesClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Cannot create k8s clientset %s", err)
		return nil
	}
	tektonClientset, err := tektonclient.NewForConfig(config)
	if err != nil {
		log.Printf("Cannot create Tekton clientset %s", err)
		return nil
	}
	tektonInformerFactory := tektoninformers.NewSharedInformerFactory(tektonClientset, resyncPeriod)
	return &clientFactory{
		kubernetesClientset:    kubernetesClientset,
		stewardClientset:       stewardClientset,
		stewardInformerFactory: stewardInformerFactory,
		tektonClientset:        tektonClientset,
		tektonInformerFactory:  tektonInformerFactory,
	}
}

// StewardInformerFactory returns Informer Factory for steward
func (f *clientFactory) StewardInformerFactory() stewardinformer.SharedInformerFactory {
	return f.stewardInformerFactory
}

// StewardV1alpha1 returns steward clients
func (f *clientFactory) StewardV1alpha1() stewardv1alpha1.StewardV1alpha1Interface {
	return f.stewardClientset.StewardV1alpha1()
}

// CoreV1 returns CoreV1 kubernetesClients
func (f *clientFactory) CoreV1() corev1.CoreV1Interface {
	return f.kubernetesClientset.CoreV1()
}

// RbacV1beta1 returns RbacV1beta1 kubernetesClients
func (f *clientFactory) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface {
	return f.kubernetesClientset.RbacV1beta1()
}

// TektonInformerFactory returns the Tekton informer factory
func (f *clientFactory) TektonInformerFactory() tektoninformers.SharedInformerFactory {
	return f.tektonInformerFactory
}

// TektonV1alpha1 returns the Tekton v1alpha1 client
func (f *clientFactory) TektonV1alpha1() tektonclientv1alpha1.TektonV1alpha1Interface {
	return f.tektonClientset.TektonV1alpha1()
}
