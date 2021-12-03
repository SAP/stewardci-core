package main

import (
	"flag"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/SAP/stewardci-core/pkg/runctl"
	"github.com/SAP/stewardci-core/pkg/signals"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"
	"knative.dev/pkg/system"
)

var kubeconfig string
var burst, qps, threadiness int

const (
	// resyncPeriod is the period between full resyncs performed
	// by the controller.
	resyncPeriod = 30 * time.Second

	// metricsPort is the TCP port number to be used by the metrics
	// HTTP server
	metricsPort = 9090
)

func init() {
	klog.InitFlags(nil)

	flag.IntVar(&burst, "burst", 10, "burst for RESTClient")
	flag.IntVar(&qps, "qps", 5, "QPS for RESTClient")
	flag.IntVar(&threadiness, "threadiness", 2, "maximum number of reconciliations performed in parallel")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.Parse()
}

//TODO: Rename "/cmd" folder to "/main"
//TODO: Rename "/cmd/controller" folder to "pipeline_run"

func main() {
	// creates the in-cluster config
	var config *rest.Config
	var err error
	defer klog.Flush()

	if kubeconfig == "" {
		klog.Infof("In cluster")
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Exitf("failed to load kubeconfig: %s; Hint: You can use parameter '-kubeconfig' for local testing", err.Error())
		}
	} else {
		klog.Infof("Outside cluster")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			klog.Exitln(err.Error())
		}
	}

	system.Namespace() // ensure that namespace is set in environment

	klog.V(3).Infof("Create Factory (resync period: %s, QPS: %d, burst: %d)", resyncPeriod.String(), qps, burst)
	config.QPS = float32(qps)
	config.Burst = burst
	factory := k8s.NewClientFactory(config, resyncPeriod)

	klog.V(2).Infof("Provide metrics on http://0.0.0.0:%d/metrics", metricsPort)
	metrics.StartServer(metricsPort)

	klog.V(3).Infof("Create Controller")
	controller := runctl.NewController(factory)

	klog.V(3).Infof("Create Signal Handlers")
	stopCh := signals.SetupShutdownSignalHandler()
	signals.SetupThreadDumpSignalHandler()

	klog.V(2).Infof("Start Informer")
	factory.StewardInformerFactory().Start(stopCh)
	factory.TektonInformerFactory().Start(stopCh)

	klog.V(2).Infof("Run controller (threadiness=%d)", threadiness)
	if err = controller.Run(threadiness, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
