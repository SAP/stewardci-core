package main

import (
	"flag"
	"log"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/SAP/stewardci-core/pkg/runctl"
	"github.com/SAP/stewardci-core/pkg/signals"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"knative.dev/pkg/system"
)

var kubeconfig string
var burst, qps int

// Time to wait until the next resync takes place.
// Resync is only required if events got lost or if the controller restarted (and missed events).
const resyncPeriod = 30 * time.Second

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	flag.IntVar(&burst, "burst", 200, "burst for RESTClient")
	flag.IntVar(&qps, "qps", 100, "QPS for RESTClient")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.Parse()
}

//TODO: Rename "/cmd" folder to "/main"
//TODO: Rename "/cmd/controller" folder to "pipeline_run"

func main() {
	// creates the in-cluster config
	var config *rest.Config
	var err error
	if kubeconfig == "" {
		log.Printf("In cluster")
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("Hint: You can use parameter '-kubeconfig' for local testing. See --help")
			panic(err.Error())
		}
	} else {
		log.Printf("Outside cluster")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	system.Namespace() // ensure that namespace is set in environment

	log.Printf("Create Factory (resync period: %s, QPS: %d, burst: %d)", resyncPeriod.String(), qps, burst)
	config.QPS = float32(qps)
	config.Burst = burst
	factory := k8s.NewClientFactory(config, resyncPeriod)

	log.Printf("Provide metrics")
	metrics := metrics.NewMetrics()
	metrics.StartServer()

	log.Printf("Create Controller")
	controller := runctl.NewController(factory, metrics)

	log.Printf("Create Signal Handler")
	stopCh := signals.SetupSignalHandler()

	log.Printf("Start Informer")
	factory.StewardInformerFactory().Start(stopCh)
	factory.TektonInformerFactory().Start(stopCh)

	log.Printf("Run controller")
	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}
