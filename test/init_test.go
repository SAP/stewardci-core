// +build e2e

package test

import (
	"log"
	"testing"
	"time"

_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"k8s.io/client-go/tools/clientcmd"
	knativetest "knative.dev/pkg/test"
)

// Time to wait until the next resync takes place.
// Resync is only required if events got lost or if the controller restarted (and missed events).
const resyncPeriod = 5 * time.Minute

func setup(t *testing.T) (k8s.ClientFactory, string) {
	t.Helper()
	kubeconfig := knativetest.Flags.Kubeconfig
	log.Printf("Create Factory (config: %s,resync period: %s)", kubeconfig, resyncPeriod.String())
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	factory := k8s.NewClientFactory(config, resyncPeriod)
	if factory == nil {
		t.Fatalf("failed to create client factory for config file '%s'.", kubeconfig)
	}

        return factory, "steward-test-c"
}
