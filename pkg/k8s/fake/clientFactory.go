package fake

import (
	"log"
	"time"

	stewardApi "github.com/SAP/stewardci-core/pkg/apis/steward"
	steward "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/fake"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/typed/steward/v1alpha1"
	stewardinformer "github.com/SAP/stewardci-core/pkg/client/informers/externalversions"
	tektonclientfake "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/fake"
	tektonclientv1alpha1 "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/typed/pipeline/v1alpha1"
	tektoninformers "github.com/SAP/stewardci-core/pkg/tektonclient/informers/externalversions"
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
)

// ClientFactory is a factory for fake clients.
type ClientFactory struct {
	kubernetesClientset    *kubernetes.Clientset
	stewardClientset       *steward.Clientset
	stewardInformerFactory stewardinformer.SharedInformerFactory
	tektonClientset        *tektonclientfake.Clientset
	tektonInformerFactory  tektoninformers.SharedInformerFactory
	sleepDuration          time.Duration
}

// NewClientFactory creates a new ClientFactory
func NewClientFactory(objects ...runtime.Object) *ClientFactory {
	stewardObjects, tektonObjects, kubernetesObjects := groupObjectsByAPI(objects)
	stewardClientset := steward.NewSimpleClientset(stewardObjects...)
	stewardInformerFactory := stewardinformer.NewSharedInformerFactory(stewardClientset, time.Minute*10)
	tektonClientset := tektonclientfake.NewSimpleClientset(tektonObjects...)
	tektonInformerFactory := tektoninformers.NewSharedInformerFactory(tektonClientset, time.Minute*10)
	sleepDuration, _ := time.ParseDuration("300ms")
	return &ClientFactory{
		kubernetesClientset:    kubernetes.NewSimpleClientset(kubernetesObjects...),
		stewardClientset:       stewardClientset,
		stewardInformerFactory: stewardInformerFactory,
		tektonClientset:        tektonClientset,
		tektonInformerFactory:  tektonInformerFactory,
		sleepDuration:          sleepDuration,
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
		case stewardApi.GroupName:
			steward = append(steward, o)
		case tektonapi.GroupName:
			tekton = append(tekton, o)
		default:
			kubernetes = append(kubernetes, o)
		}
	}
	return
}

// StewardV1alpha1 returns Steward clients.
func (f *ClientFactory) StewardV1alpha1() stewardv1alpha1.StewardV1alpha1Interface {
	return f.stewardClientset.StewardV1alpha1()
}

// StewardClientset returns Steward clientset.
func (f *ClientFactory) StewardClientset() *steward.Clientset {
	return f.stewardClientset
}

// StewardInformerFactory returns the informer factory for Steward
func (f *ClientFactory) StewardInformerFactory() stewardinformer.SharedInformerFactory {
	return f.stewardInformerFactory
}

// KubernetesClientset returns the Kubernetes fake clientset.
func (f *ClientFactory) KubernetesClientset() *kubernetes.Clientset {
	return f.kubernetesClientset
}

// CoreV1 returns fake CoreV1 clients
func (f *ClientFactory) CoreV1() corev1.CoreV1Interface {
	return f.kubernetesClientset.CoreV1()
}

// RbacV1beta1 returns fake RbacV1beta1 clients
func (f *ClientFactory) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface {
	return f.kubernetesClientset.RbacV1beta1()
}

// TektonInformerFactory returns the Tekton informer factory
func (f *ClientFactory) TektonInformerFactory() tektoninformers.SharedInformerFactory {
	return f.tektonInformerFactory
}

// TektonV1alpha1 returns the Tekton v1alpha1 client
func (f *ClientFactory) TektonV1alpha1() tektonclientv1alpha1.TektonV1alpha1Interface {
	return f.tektonClientset.TektonV1alpha1()
}

// Sleep sleeps and logs the start and the end of the sleep.
func (f *ClientFactory) Sleep(message string) {
	log.Printf("Sleep start: %s", message)
	time.Sleep(f.sleepDuration)
	log.Printf("Sleep end: %s", message)
}

// CheckTimeOrder checks if the duration between start and end is at least one sleep duration long.
func (f *ClientFactory) CheckTimeOrder(start metav1.Time, end metav1.Time) bool {
	return end.After(start.Add(f.sleepDuration))
}
