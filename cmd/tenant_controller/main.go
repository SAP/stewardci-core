package main

import (
	"flag"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/signals"
	tenantctl "github.com/SAP/stewardci-core/pkg/tenantctl"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"
	"knative.dev/pkg/system"
)

var kubeconfig string

// Time to wait until the next resync takes place.
// Resync is only required if events got lost or if the controller restarted (and missed events).
const resyncPeriod = 1 * time.Minute

func init() {
	klog.InitFlags(nil)

	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.Parse()
}

func main() {
	// creates the in-cluster config
	var config *rest.Config
	var err error
	defer klog.Flush()

	if kubeconfig == "" {
		klog.Infof("In cluster")
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Infof("Hint: You can use parameter '-kubeconfig' for local testing. See --help")
			panic(err.Error())
		}
	} else {
		klog.Infof("Outside cluster")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	system.Namespace() // ensure that namespace is set in environment

	klog.V(3).Infof("Create Factory (resync period: %s)", resyncPeriod.String())
	factory := k8s.NewClientFactory(config, resyncPeriod)

	klog.V(2).Infof("Provide metrics")
	metrics := tenantctl.NewMetrics()
	metrics.StartServer()

	klog.V(3).Infof("Create Controller")
	controller := tenantctl.NewController(factory, metrics)

	klog.V(3).Infof("Create Signal Handler")
	stopCh := signals.SetupSignalHandler()

	klog.V(2).Infof("Start Informer")
	factory.StewardInformerFactory().Start(stopCh)

	klog.V(2).Infof("Run controller")
	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
