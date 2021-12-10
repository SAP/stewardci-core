package fake

import (
	"time"

	stewardapis "github.com/SAP/stewardci-core/pkg/apis/steward"
	stewardclientfake "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/fake"
	stewardv1alpha1client "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	stewardinformer "github.com/SAP/stewardci-core/pkg/client/informers/externalversions"
	tektonclientfake "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/fake"
	tektonv1beta1client "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/typed/pipeline/v1beta1"
	tektoninformers "github.com/SAP/stewardci-core/pkg/tektonclient/informers/externalversions"
	tektonapis "github.com/tektoncd/pipeline/pkg/apis/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	dynamic "k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sclientfake "k8s.io/client-go/kubernetes/fake"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingv1client "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	klog "k8s.io/klog/v2"
)

// ClientFactory is a factory for fake clients.
type ClientFactory struct {
	kubernetesClientset    *k8sclientfake.Clientset
	DynamicClient          *dynamicfake.FakeDynamicClient
	stewardClientset       *stewardclientfake.Clientset
	stewardInformerFactory stewardinformer.SharedInformerFactory
	tektonClientset        *tektonclientfake.Clientset
	tektonInformerFactory  tektoninformers.SharedInformerFactory
	sleepDuration          time.Duration
}

// NewClientFactory creates a new ClientFactory
func NewClientFactory(objects ...runtime.Object) *ClientFactory {
	stewardObjects, tektonObjects, kubernetesObjects := groupObjectsByAPI(objects)
	stewardClientset := stewardclientfake.NewSimpleClientset(stewardObjects...)
	stewardInformerFactory := stewardinformer.NewSharedInformerFactory(stewardClientset, 10*time.Minute)
	tektonClientset := tektonclientfake.NewSimpleClientset(tektonObjects...)
	tektonInformerFactory := tektoninformers.NewSharedInformerFactory(tektonClientset, 10*time.Minute)

	return &ClientFactory{
		kubernetesClientset:    k8sclientfake.NewSimpleClientset(kubernetesObjects...),
		DynamicClient:          dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		stewardClientset:       stewardClientset,
		stewardInformerFactory: stewardInformerFactory,
		tektonClientset:        tektonClientset,
		tektonInformerFactory:  tektonInformerFactory,
		sleepDuration:          300 * time.Millisecond,
	}
}

func groupObjectsByAPI(objects []runtime.Object) (
	steward []runtime.Object,
	tekton []runtime.Object,
	kubernetes []runtime.Object,
) {
	steward = []runtime.Object{}
	tekton = []runtime.Object{}
	kubernetes = []runtime.Object{}
	for _, o := range objects {
		switch o.GetObjectKind().GroupVersionKind().Group {
		case stewardapis.GroupName:
			steward = append(steward, o)
		case tektonapis.GroupName:
			tekton = append(tekton, o)
		default:
			kubernetes = append(kubernetes, o)
		}
	}
	return
}

// StewardClientset returns the Steward fake clientset.
func (f *ClientFactory) StewardClientset() *stewardclientfake.Clientset {
	return f.stewardClientset
}

// StewardV1alpha1 implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) StewardV1alpha1() stewardv1alpha1client.StewardV1alpha1Interface {
	return f.stewardClientset.StewardV1alpha1()
}

// StewardInformerFactory implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) StewardInformerFactory() stewardinformer.SharedInformerFactory {
	return f.stewardInformerFactory
}

// KubernetesClientset returns the Kubernetes fake clientset.
func (f *ClientFactory) KubernetesClientset() *k8sclientfake.Clientset {
	return f.kubernetesClientset
}

// CoreV1 implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) CoreV1() corev1client.CoreV1Interface {
	return f.kubernetesClientset.CoreV1()
}

// Dynamic implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) Dynamic() dynamic.Interface {
	return f.DynamicClient
}

// DynamicFake returns the dynamic Kubernetes fake client.
func (f *ClientFactory) DynamicFake() *dynamicfake.FakeDynamicClient {
	return f.DynamicClient
}

// NetworkingV1 implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) NetworkingV1() networkingv1client.NetworkingV1Interface {
	return f.kubernetesClientset.NetworkingV1()
}

// RbacV1 implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) RbacV1() rbacv1client.RbacV1Interface {
	return f.kubernetesClientset.RbacV1()
}

// TektonInformerFactory implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) TektonInformerFactory() tektoninformers.SharedInformerFactory {
	return f.tektonInformerFactory
}

// TektonClientset returns the Tekton fake clientset.
func (f *ClientFactory) TektonClientset() *tektonclientfake.Clientset {
	return f.tektonClientset
}

// TektonV1beta1 implements interface "github.com/SAP/stewardci-core/pkg/k8s".ClientFactory
func (f *ClientFactory) TektonV1beta1() tektonv1beta1client.TektonV1beta1Interface {
	return f.tektonClientset.TektonV1beta1()
}

// Sleep sleeps and logs the start and the end of the sleep.
func (f *ClientFactory) Sleep(message string) {
	klog.Infof("Sleep start: %s", message)
	time.Sleep(f.sleepDuration)
	klog.Infof("Sleep end: %s", message)
}

// CheckTimeOrder checks if the duration between start and end is at least one sleep duration long.
func (f *ClientFactory) CheckTimeOrder(start metav1.Time, end metav1.Time) bool {
	return end.After(start.Add(f.sleepDuration))
}
