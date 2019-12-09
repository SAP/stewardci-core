package main

import (
	"flag"
	"log"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/signals"
	tenantctl "github.com/SAP/stewardci-core/pkg/tenantctl"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig string

// Time to wait until the next resync takes place.
// Resync is only required if events got lost or if the controller restarted (and missed events).
const resyncPeriod = 1 * time.Minute

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)

	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.Parse()
}

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
	log.Printf("Create Factory (resync period: %s)", resyncPeriod.String())
	factory := k8s.NewClientFactory(config, resyncPeriod)

	log.Printf("Provide metrics")
	metrics := tenantctl.NewMetrics()
	metrics.StartServer()

	log.Printf("Create Controller")
	controller := tenantctl.NewController(factory, metrics)

	log.Printf("Create Signal Handler")
	stopCh := signals.SetupSignalHandler()

	log.Printf("Start Informer")
	factory.StewardInformerFactory().Start(stopCh)

	log.Printf("Run controller")
	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}
